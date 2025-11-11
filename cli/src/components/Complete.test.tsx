import { describe, it, expect } from 'vitest';
import React from 'react';
import { render } from 'ink-testing-library';
import { Complete } from './Complete.js';
import type { Configuration } from './ConfigurationWizard.js';

describe('Complete component', () => {
  const mockConfig: Configuration = {
    namespace: 'test-namespace',
    releaseName: 'test-release',
    ingressHost: 'supacontrol.example.com',
    tlsEnabled: false,
    certManagerIssuer: '',
    useExternalDb: false,
    dbPassword: 'test-password',
    jwtSecret: 'test-jwt-secret',
  };

  describe('when installation is successful', () => {
    it('should render success message', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('Installation Successful');
      expect(output).toContain('SupaControl is now running on your cluster');
    });

    it('should display access information', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('Access Information');
      expect(output).toContain('Dashboard URL');
      expect(output).toContain('http://supacontrol.example.com');
      expect(output).toContain('Namespace');
      expect(output).toContain('test-namespace');
      expect(output).toContain('Release');
      expect(output).toContain('test-release');
    });

    it('should display default credentials', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('Default Credentials');
      expect(output).toContain('Username');
      expect(output).toContain('admin');
      expect(output).toContain('Password');
      expect(output).toContain('Change the default password immediately');
    });

    it('should display next steps', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('Next Steps');
      expect(output).toContain('Wait for all pods to be ready');
      expect(output).toContain('kubectl get pods');
      expect(output).toContain('Access the dashboard');
      expect(output).toContain('Login with default credentials');
      expect(output).toContain('Generate an API key');
      expect(output).toContain('Create your first Supabase instance');
    });

    it('should display useful commands', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('Useful Commands');
      expect(output).toContain('View logs');
      expect(output).toContain('Check status');
      expect(output).toContain('Port forward');
    });

    it('should use HTTPS URL when TLS is enabled', () => {
      const tlsConfig = { ...mockConfig, tlsEnabled: true };
      const { lastFrame } = render(<Complete config={tlsConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('https://supacontrol.example.com');
    });

    it('should display TLS note when TLS is enabled', () => {
      const tlsConfig = { ...mockConfig, tlsEnabled: true };
      const { lastFrame } = render(<Complete config={tlsConfig} success={true} />);
      const output = lastFrame();

      expect(output).toContain('Note about TLS/HTTPS');
      expect(output).toContain('cert-manager');
    });

    it('should not display TLS note when TLS is disabled', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={true} />);
      const output = lastFrame();

      expect(output).not.toContain('Note about TLS/HTTPS');
    });
  });

  describe('when installation fails', () => {
    it('should render failure message', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={false} />);
      const output = lastFrame();

      expect(output).toContain('Installation Failed');
    });

    it('should display troubleshooting steps', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={false} />);
      const output = lastFrame();

      expect(output).toContain('Check the error messages above');
      expect(output).toContain('Verify your Kubernetes cluster is accessible');
      expect(output).toContain('Ensure you have proper permissions');
      expect(output).toContain('Try running the installer again');
    });

    it('should display help link', () => {
      const { lastFrame } = render(<Complete config={mockConfig} success={false} />);
      const output = lastFrame();

      expect(output).toContain('For help, visit');
      expect(output).toContain('github.com/qubitquilt/SupaControl');
    });
  });
});
