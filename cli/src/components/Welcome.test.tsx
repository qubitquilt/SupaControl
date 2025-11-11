import { describe, it, expect } from 'vitest';
import React from 'react';
import { render } from 'ink-testing-library';
import { Welcome } from './Welcome.js';

describe('Welcome component', () => {
  it('should render welcome screen with title', () => {
    const { lastFrame } = render(<Welcome />);
    const output = lastFrame();

    expect(output).toBeTruthy();
    expect(output).toContain('Supabase Management Platform Installer');
  });

  it('should display installation items', () => {
    const { lastFrame } = render(<Welcome />);
    const output = lastFrame();

    expect(output).toContain('What will be installed:');
    expect(output).toContain('SupaControl API Server');
    expect(output).toContain('PostgreSQL Database');
    expect(output).toContain('Web Dashboard');
    expect(output).toContain('Kubernetes RBAC Configuration');
    expect(output).toContain('Ingress for HTTPS access');
  });

  it('should show continue instruction', () => {
    const { lastFrame } = render(<Welcome />);
    const output = lastFrame();

    expect(output).toContain('Press any key to continue');
  });
});
