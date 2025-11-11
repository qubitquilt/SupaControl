import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { render } from 'ink-testing-library';
import { Installation } from './Installation.js';
import type { Configuration } from './ConfigurationWizard.js';
import * as helm from '../utils/helm.js';
import { access } from 'fs/promises';

// Mock dependencies
vi.mock('../utils/helm.js', () => ({
  saveHelmValues: vi.fn(),
  installHelm: vi.fn(),
  checkHelmRelease: vi.fn(),
  checkPodStatus: vi.fn(),
}));

vi.mock('fs/promises', () => ({
  access: vi.fn(),
  constants: { R_OK: 4 },
}));

vi.mock('execa', () => ({
  execa: vi.fn(),
}));

describe('Installation component', () => {
  const mockConfig: Configuration = {
    namespace: 'test-namespace',
    releaseName: 'test-release',
    ingressHost: 'supacontrol.example.com',
    ingressDomain: 'supabase.example.com',
    ingressClass: 'nginx',
    tlsEnabled: false,
    certManagerIssuer: '',
    installDatabase: true,
    dbPassword: 'test-password',
    jwtSecret: 'test-jwt-secret',
  };

  const mockOnComplete = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render installation screen', () => {
    vi.mocked(helm.saveHelmValues).mockImplementation(() => new Promise(() => {}));
    vi.mocked(access).mockImplementation(() => new Promise(() => {}));

    const { lastFrame } = render(<Installation config={mockConfig} onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('Installing SupaControl');
  });

  it('should show installation steps', () => {
    vi.mocked(helm.saveHelmValues).mockImplementation(() => new Promise(() => {}));
    vi.mocked(access).mockImplementation(() => new Promise(() => {}));

    const { lastFrame } = render(<Installation config={mockConfig} onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('Initializing installation');
    expect(output).toContain('Generating Helm values');
  });

  it('should handle successful installation', async () => {
    vi.mocked(access).mockResolvedValue(undefined);
    vi.mocked(helm.saveHelmValues).mockResolvedValue('/path/to/values.yaml');
    vi.mocked(helm.installHelm).mockResolvedValue({
      success: true,
      output: 'Installation successful',
      action: 'install',
    });
    vi.mocked(helm.checkHelmRelease).mockResolvedValue(true);
    vi.mocked(helm.checkPodStatus).mockResolvedValue({
      success: true,
      pods: [],
      errors: [],
    });

    render(<Installation config={mockConfig} onComplete={mockOnComplete} />);

    await new Promise(resolve => setTimeout(resolve, 500));

    expect(helm.saveHelmValues).toHaveBeenCalled();
  });

  it('should handle installation errors', async () => {
    vi.mocked(access).mockResolvedValue(undefined);
    vi.mocked(helm.saveHelmValues).mockResolvedValue('/path/to/values.yaml');
    vi.mocked(helm.installHelm).mockResolvedValue({
      success: false,
      error: 'Installation failed',
    });

    render(<Installation config={mockConfig} onComplete={mockOnComplete} />);

    await new Promise(resolve => setTimeout(resolve, 500));

    expect(helm.saveHelmValues).toHaveBeenCalled();
  });

  it('should call onComplete when installation finishes', async () => {
    vi.mocked(access).mockResolvedValue(undefined);
    vi.mocked(helm.saveHelmValues).mockResolvedValue('/path/to/values.yaml');
    vi.mocked(helm.installHelm).mockResolvedValue({
      success: true,
      output: 'Installation successful',
      action: 'install',
    });
    vi.mocked(helm.checkHelmRelease).mockResolvedValue(true);
    vi.mocked(helm.checkPodStatus).mockResolvedValue({
      success: true,
      pods: [],
      errors: [],
    });

    render(<Installation config={mockConfig} onComplete={mockOnComplete} />);

    // Wait for all async operations
    await new Promise(resolve => setTimeout(resolve, 2000));

    expect(mockOnComplete).toHaveBeenCalled();
  });
});
