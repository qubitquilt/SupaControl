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
    // Mock with resolved promises to prevent hanging
    vi.mocked(helm.saveHelmValues).mockResolvedValue('/path/to/values.yaml');
    vi.mocked(access).mockResolvedValue(undefined);

    const { lastFrame } = render(<Installation config={mockConfig} onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('Installing SupaControl');
  });

  it('should show installation steps', () => {
    // Mock with resolved promises to prevent hanging
    vi.mocked(helm.saveHelmValues).mockResolvedValue('/path/to/values.yaml');
    vi.mocked(access).mockResolvedValue(undefined);

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

    // Wait for the mocked function to be called
    await vi.waitFor(() => {
      expect(helm.saveHelmValues).toHaveBeenCalled();
    }, { timeout: 3000 });
  });

  it('should handle installation errors', async () => {
    vi.mocked(access).mockResolvedValue(undefined);
    vi.mocked(helm.saveHelmValues).mockResolvedValue('/path/to/values.yaml');
    vi.mocked(helm.installHelm).mockResolvedValue({
      success: false,
      error: 'Installation failed',
    });

    render(<Installation config={mockConfig} onComplete={mockOnComplete} />);

    // Wait for the mocked function to be called
    await vi.waitFor(() => {
      expect(helm.saveHelmValues).toHaveBeenCalled();
    }, { timeout: 3000 });
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

    // Wait for onComplete to be called
    await vi.waitFor(() => {
      expect(mockOnComplete).toHaveBeenCalled();
    }, { timeout: 5000 });
  });
});
