import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { render } from 'ink-testing-library';
import { PrerequisitesCheck } from './PrerequisitesCheck.js';
import * as prerequisites from '../utils/prerequisites.js';

// Mock the prerequisites module
vi.mock('../utils/prerequisites.js', () => ({
  checkPrerequisites: vi.fn(),
  checkKubernetesConnection: vi.fn(),
}));

describe('PrerequisitesCheck component', () => {
  const mockOnComplete = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should show checking message initially', () => {
    vi.mocked(prerequisites.checkPrerequisites).mockImplementation(() =>
      new Promise(() => {}) // Never resolves to keep in checking state
    );
    vi.mocked(prerequisites.checkKubernetesConnection).mockImplementation(() =>
      new Promise(() => {})
    );

    const { lastFrame } = render(<PrerequisitesCheck onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('Checking Prerequisites');
    expect(output).toContain('Checking system requirements');
  });

  it('should display prerequisite results when all pass', async () => {
    const mockResults = [
      { name: 'kubectl', installed: true, version: 'v1.28.0', required: true, installUrl: '' },
      { name: 'helm', installed: true, version: 'v3.12.0', required: true, installUrl: '' },
    ];

    vi.mocked(prerequisites.checkPrerequisites).mockResolvedValue(mockResults);
    vi.mocked(prerequisites.checkKubernetesConnection).mockResolvedValue(true);

    const { lastFrame } = render(<PrerequisitesCheck onComplete={mockOnComplete} />);

    // Wait for async operations
    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame();

    expect(output).toContain('kubectl');
    expect(output).toContain('v1.28.0');
    expect(output).toContain('helm');
    expect(output).toContain('v3.12.0');
    expect(output).toContain('Kubernetes Cluster Connection');
    expect(output).toContain('Connected');
    expect(output).toContain('All prerequisites met');
  });

  it('should display missing prerequisites', async () => {
    const mockResults = [
      { name: 'kubectl', installed: true, version: 'v1.28.0', required: true, installUrl: '' },
      {
        name: 'helm',
        installed: false,
        required: true,
        installUrl: 'https://helm.sh/docs/intro/install/'
      },
    ];

    vi.mocked(prerequisites.checkPrerequisites).mockResolvedValue(mockResults);
    vi.mocked(prerequisites.checkKubernetesConnection).mockResolvedValue(true);

    const { lastFrame } = render(<PrerequisitesCheck onComplete={mockOnComplete} />);

    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame();

    expect(output).toContain('kubectl');
    expect(output).toContain('helm');
    expect(output).toContain('Not found');
    expect(output).toContain('Required');
    expect(output).toContain('Some prerequisites are missing');
  });

  it('should show Kubernetes connection failure', async () => {
    const mockResults = [
      { name: 'kubectl', installed: true, version: 'v1.28.0', required: true, installUrl: '' },
    ];

    vi.mocked(prerequisites.checkPrerequisites).mockResolvedValue(mockResults);
    vi.mocked(prerequisites.checkKubernetesConnection).mockResolvedValue(false);

    const { lastFrame } = render(<PrerequisitesCheck onComplete={mockOnComplete} />);

    await new Promise(resolve => setTimeout(resolve, 100));

    const output = lastFrame();

    expect(output).toContain('Kubernetes Cluster Connection');
    expect(output).toContain('Not connected');
    expect(output).toContain('Check kubectl configuration');
  });

  it('should call onComplete with true when all prerequisites pass', async () => {
    const mockResults = [
      { name: 'kubectl', installed: true, version: 'v1.28.0', required: true, installUrl: '' },
      { name: 'helm', installed: true, version: 'v3.12.0', required: true, installUrl: '' },
    ];

    vi.mocked(prerequisites.checkPrerequisites).mockResolvedValue(mockResults);
    vi.mocked(prerequisites.checkKubernetesConnection).mockResolvedValue(true);

    render(<PrerequisitesCheck onComplete={mockOnComplete} />);

    // Wait for async operations and timeout
    await new Promise(resolve => setTimeout(resolve, 1100));

    expect(mockOnComplete).toHaveBeenCalledWith(true);
  });

  it('should call onComplete with false when prerequisites are missing', async () => {
    const mockResults = [
      {
        name: 'helm',
        installed: false,
        required: true,
        installUrl: 'https://helm.sh/docs/intro/install/'
      },
    ];

    vi.mocked(prerequisites.checkPrerequisites).mockResolvedValue(mockResults);
    vi.mocked(prerequisites.checkKubernetesConnection).mockResolvedValue(true);

    render(<PrerequisitesCheck onComplete={mockOnComplete} />);

    await new Promise(resolve => setTimeout(resolve, 1100));

    expect(mockOnComplete).toHaveBeenCalledWith(false);
  });

  it('should call onComplete with false when Kubernetes is not connected', async () => {
    const mockResults = [
      { name: 'kubectl', installed: true, version: 'v1.28.0', required: true, installUrl: '' },
    ];

    vi.mocked(prerequisites.checkPrerequisites).mockResolvedValue(mockResults);
    vi.mocked(prerequisites.checkKubernetesConnection).mockResolvedValue(false);

    render(<PrerequisitesCheck onComplete={mockOnComplete} />);

    await new Promise(resolve => setTimeout(resolve, 1100));

    expect(mockOnComplete).toHaveBeenCalledWith(false);
  });
});
