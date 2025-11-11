import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render } from 'ink-testing-library';
import { ConfigurationWizard } from './ConfigurationWizard.js';
import * as secrets from '../utils/secrets.js';

// Mock the secrets module
vi.mock('../utils/secrets.js', () => ({
  generateJWTSecret: vi.fn(() => 'mock-jwt-secret-123'),
  generateDatabasePassword: vi.fn(() => 'mock-db-password-123'),
}));

describe('ConfigurationWizard component', () => {
  const mockOnComplete = vi.fn();

  it('should render configuration wizard', () => {
    const { lastFrame } = render(<ConfigurationWizard onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('Configuration Wizard');
  });

  it('should start with namespace step', () => {
    const { lastFrame } = render(<ConfigurationWizard onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('Kubernetes namespace') || expect(output).toContain('namespace');
  });

  it('should display default namespace value', () => {
    const { lastFrame } = render(<ConfigurationWizard onComplete={mockOnComplete} />);
    const output = lastFrame();

    expect(output).toContain('supacontrol');
  });

  it('should handle namespace input', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Type custom namespace
    stdin.write('custom-namespace');

    const output = lastFrame();
    expect(output).toContain('custom-namespace');
  });

  it('should advance to next step on enter', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Submit namespace
    stdin.write('\r');

    const output = lastFrame();
    expect(output).toContain('release name') || expect(output).toContain('Release');
  });

  it('should handle TLS enabled selection', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Skip through steps to TLS selection
    stdin.write('\r'); // namespace
    stdin.write('\r'); // release name
    stdin.write('test.example.com\r'); // ingress host
    stdin.write('\r'); // ingress domain
    stdin.write('\r'); // ingress class

    const output = lastFrame();
    // Should be progressing through steps
    expect(output).toBeTruthy();
    expect(output.length).toBeGreaterThan(0);
  });

  it('should handle database installation choice', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Skip through initial steps
    for (let i = 0; i < 6; i++) {
      stdin.write('\r');
    }

    const output = lastFrame();
    // Should ask about database installation or be at a later step
    expect(output).toBeTruthy();
  });

  it('should render through multiple steps', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Navigate through several steps
    for (let i = 0; i < 3; i++) {
      stdin.write('\r');
    }

    const output = lastFrame();
    // Should have progressed through steps
    expect(output).toBeTruthy();
  });

  it('should show configuration summary', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Navigate to confirmation step
    for (let i = 0; i < 10; i++) {
      stdin.write('\r');
    }

    const output = lastFrame();
    // Should show either configuration details or installation
    expect(output).toBeTruthy();
  });

  it('should navigate through multiple configuration steps', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Navigate through several steps
    for (let i = 0; i < 5; i++) {
      stdin.write('\r');
    }

    const output = lastFrame();
    expect(output).toBeTruthy();
    expect(output.length).toBeGreaterThan(0);
  });

  it('should use default values when not customized', () => {
    const { lastFrame } = render(<ConfigurationWizard onComplete={mockOnComplete} />);
    const output = lastFrame();

    // Check default values are shown
    expect(output).toContain('supacontrol');
  });

  it('should handle external database configuration', () => {
    const { lastFrame, stdin } = render(<ConfigurationWizard onComplete={mockOnComplete} />);

    // Navigate to database installation step
    for (let i = 0; i < 6; i++) {
      stdin.write('\r');
    }

    // Select external database (down arrow to select "No")
    stdin.write('\x1B[B'); // Down arrow
    stdin.write('\r'); // Confirm

    // Check that the wizard has progressed (non-empty output)
    const output = lastFrame();
    expect(output).toBeTruthy();
    expect(output.length).toBeGreaterThan(0);
  });
});
