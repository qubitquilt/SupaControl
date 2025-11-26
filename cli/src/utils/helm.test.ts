import { describe, it, expect, vi, beforeEach } from 'vitest';
import { parse } from 'yaml';
import { execa } from 'execa';
import {
  generateHelmValues,
  saveHelmValues,
  installHelm,
  upgradeHelm,
  checkHelmRelease,
  uninstallHelm,
  checkPodStatus,
  checkHelmConnection,
} from './helm.js';
import type { HelmConfig } from './helm.js';

// Mock dependencies
vi.mock('execa');
vi.mock('fs/promises', () => ({
  writeFile: vi.fn(),
  mkdir: vi.fn(),
}));

describe('helm utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('generateHelmValues', () => {
    it('should generate basic helm values with required fields', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: false,
        tlsEnabled: false,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.config.jwtSecret).toBe('test-jwt-secret');
      expect(parsed.config.database.password).toBe('test-db-password');
      expect(parsed.replicaCount).toBe(1);
    });

    it('should use default values for optional fields', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: false,
        tlsEnabled: false,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.config.database.host).toBe('supacontrol-postgresql');
      expect(parsed.config.database.port).toBe('5432');
      expect(parsed.config.database.user).toBe('supacontrol');
      expect(parsed.config.database.name).toBe('supacontrol');
    });

    it('should configure PostgreSQL when no external database is specified', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: false,
        tlsEnabled: false,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.postgresql.enabled).toBe(true);
      expect(parsed.postgresql.auth.password).toBe('test-db-password');
      expect(parsed.postgresql.primary.persistence.enabled).toBe(true);
    });

    it('should disable PostgreSQL when external database is specified', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        dbHost: 'external-db.example.com',
        ingressEnabled: false,
        tlsEnabled: false,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.postgresql.enabled).toBe(false);
      expect(parsed.config.database.host).toBe('external-db.example.com');
    });

    it('should configure ingress when enabled', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: true,
        ingressHost: 'supacontrol.example.com',
        ingressClass: 'nginx',
        tlsEnabled: false,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.ingress.enabled).toBe(true);
      expect(parsed.ingress.className).toBe('nginx');
      expect(parsed.ingress.hosts[0].host).toBe('supacontrol.example.com');
      expect(parsed.ingress.hosts[0].paths[0].path).toBe('/');
    });

    it('should configure TLS when enabled', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: true,
        ingressHost: 'supacontrol.example.com',
        tlsEnabled: true,
        certManagerIssuer: 'letsencrypt-prod',
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.ingress.tls).toBeDefined();
      expect(parsed.ingress.tls[0].secretName).toBe('supacontrol-tls');
      expect(parsed.ingress.tls[0].hosts).toContain('supacontrol.example.com');
      expect(parsed.ingress.annotations['cert-manager.io/cluster-issuer']).toBe('letsencrypt-prod');
    });

    it('should not add cert-manager annotation when TLS is disabled', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: true,
        ingressHost: 'supacontrol.example.com',
        tlsEnabled: false,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.ingress.annotations).toEqual({});
      expect(parsed.ingress.tls).toBeUndefined();
    });

    it('should use custom certManagerIssuer when provided', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: true,
        ingressHost: 'supacontrol.example.com',
        tlsEnabled: true,
        certManagerIssuer: 'letsencrypt-staging',
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.ingress.annotations['cert-manager.io/cluster-issuer']).toBe('letsencrypt-staging');
    });

    it('should set custom replica count', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: false,
        tlsEnabled: false,
        replicas: 3,
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.replicaCount).toBe(3);
    });

    it('should set custom image tag', () => {
      const config: HelmConfig = {
        jwtSecret: 'test-jwt-secret',
        dbPassword: 'test-db-password',
        ingressEnabled: false,
        tlsEnabled: false,
        imageTag: 'v1.2.3',
      };

      const values = generateHelmValues(config);
      const parsed = parse(values);

      expect(parsed.image.tag).toBe('v1.2.3');
    });
  });

  describe('installHelm', () => {
    it('should call helm install with correct arguments', async () => {
      // Mock checkHelmRelease to return false (release doesn't exist)
      vi.mocked(execa)
        .mockRejectedValueOnce(new Error('Error: release: not found')) // checkHelmRelease call
        .mockResolvedValueOnce({
          stdout: 'v3.12.0',
          stderr: '',
          exitCode: 0,
        } as any) // checkHelmConnection call
        .mockResolvedValueOnce({
          stdout: 'dry-run output',
          stderr: '',
          exitCode: 0,
        } as any) // dry-run call
        .mockResolvedValueOnce({
          stdout: 'Release "supacontrol" deployed',
          stderr: '',
          exitCode: 0,
        } as any); // actual install call

      const result = await installHelm(
        'test-namespace',
        'test-release',
        '/path/to/values.yaml',
        '/path/to/chart'
      );

      expect(result.success).toBe(true);
      expect(result.output).toContain('deployed');
      // Verify the actual install call (4th call)
      expect(execa).toHaveBeenNthCalledWith(4, 'helm', [
        'install',
        'test-release',
        '/path/to/chart',
        '--namespace',
        'test-namespace',
        '--create-namespace',
        '--values',
        '/path/to/values.yaml',
        '--wait',
        '--debug',
      ], expect.objectContaining({ timeout: 120000 }));
    });

    it('should return error when helm install fails', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Chart not found'));

      const result = await installHelm(
        'test-namespace',
        'test-release',
        '/path/to/values.yaml',
        '/path/to/chart'
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain('Chart not found');
    });
  });

  describe('upgradeHelm', () => {
    it('should call helm upgrade with correct arguments', async () => {
      // Mock dry-run call first, then actual upgrade
      vi.mocked(execa)
        .mockResolvedValueOnce({
          stdout: 'dry-run output',
          stderr: '',
          exitCode: 0,
        } as any) // dry-run call
        .mockResolvedValueOnce({
          stdout: 'Release "supacontrol" upgraded',
          stderr: '',
          exitCode: 0,
        } as any); // actual upgrade call

      const result = await upgradeHelm(
        'test-namespace',
        'test-release',
        '/path/to/values.yaml',
        '/path/to/chart'
      );

      expect(result.success).toBe(true);
      expect(result.output).toContain('upgraded');
      // Verify the actual upgrade call (2nd call)
      expect(execa).toHaveBeenNthCalledWith(2, 'helm', [
        'upgrade',
        'test-release',
        '/path/to/chart',
        '--namespace',
        'test-namespace',
        '--values',
        '/path/to/values.yaml',
        '--wait',
        '--debug',
      ], expect.objectContaining({ timeout: 70000 }));
    });

    it('should return error when helm upgrade fails', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Release not found'));

      const result = await upgradeHelm(
        'test-namespace',
        'test-release',
        '/path/to/values.yaml',
        '/path/to/chart'
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain('Release not found');
    });
  });

  describe('checkHelmRelease', () => {
    it('should return true when release exists', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: '{"status": "deployed"}',
        stderr: '',
        exitCode: 0,
      } as any);

      const result = await checkHelmRelease('test-namespace', 'test-release');

      expect(result).toBe(true);
      expect(execa).toHaveBeenCalledWith('helm', [
        'status',
        'test-release',
        '--namespace',
        'test-namespace',
        '--output',
        'json',
      ], expect.objectContaining({ timeout: 30000 }));
    });

    it('should return false when release does not exist', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Release not found'));

      const result = await checkHelmRelease('test-namespace', 'test-release');

      expect(result).toBe(false);
    });
  });

  describe('uninstallHelm', () => {
    it('should call helm uninstall with correct arguments', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'release "test-release" uninstalled',
        stderr: '',
        exitCode: 0,
      } as any);

      const result = await uninstallHelm('test-namespace', 'test-release');

      expect(result.success).toBe(true);
      expect(result.output).toContain('uninstalled');
      expect(execa).toHaveBeenCalledWith('helm', [
        'uninstall',
        'test-release',
        '--namespace',
        'test-namespace',
      ]);
    });

    it('should return error when helm uninstall fails', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Release not found'));

      const result = await uninstallHelm('test-namespace', 'test-release');

      expect(result.success).toBe(false);
      expect(result.error).toContain('Release not found');
    });
  });

  describe('checkPodStatus', () => {
    it('should return pod status when pods exist', async () => {
      const mockPodData = {
        items: [
          {
            metadata: { name: 'test-pod-1' },
            status: {
              phase: 'Running',
              startTime: '2023-01-01T00:00:00Z',
              containerStatuses: [
                { ready: true, restartCount: 0, name: 'container-1', state: { running: {} } },
              ],
            },
            spec: {
              containers: [{ name: 'container-1' }],
            },
          },
        ],
      };

      vi.mocked(execa).mockResolvedValue({
        stdout: JSON.stringify(mockPodData),
        stderr: '',
        exitCode: 0,
      } as any);

      const result = await checkPodStatus('test-namespace', 'test-release');

      expect(result.success).toBe(true);
      expect(result.pods).toHaveLength(1);
      expect(result.pods[0].name).toBe('test-pod-1');
      expect(result.pods[0].status).toBe('Running');
      expect(result.errors).toHaveLength(0);
    });

    it('should detect image pull errors', async () => {
      const mockPodData = {
        items: [
          {
            metadata: { name: 'test-pod-1' },
            status: {
              phase: 'Pending',
              startTime: '2023-01-01T00:00:00Z',
              containerStatuses: [
                {
                  ready: false,
                  restartCount: 0,
                  name: 'container-1',
                  state: {
                    waiting: {
                      reason: 'ImagePullBackOff',
                      message: 'Failed to pull image',
                    },
                  },
                },
              ],
            },
            spec: {
              containers: [{ name: 'container-1' }],
            },
          },
        ],
      };

      vi.mocked(execa).mockResolvedValue({
        stdout: JSON.stringify(mockPodData),
        stderr: '',
        exitCode: 0,
      } as any);

      const result = await checkPodStatus('test-namespace', 'test-release');

      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
      expect(result.errors[0]).toContain('Failed to pull image');
    });

    it('should return error when kubectl fails', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('kubectl error'));

      const result = await checkPodStatus('test-namespace', 'test-release');

      expect(result.success).toBe(false);
      expect(result.pods).toHaveLength(0);
      expect(result.errors[0]).toContain('kubectl error');
    });
  });

  describe('checkHelmConnection', () => {
    it('should return working true when helm is available', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'v3.12.0',
        stderr: '',
        exitCode: 0,
      } as any);

      const result = await checkHelmConnection();

      expect(result.working).toBe(true);
      expect(execa).toHaveBeenCalledWith('helm', ['version', '--short'], expect.objectContaining({ timeout: 10000 }));
    });

    it('should return working false when helm is not available', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('helm not found'));

      const result = await checkHelmConnection();

      expect(result.working).toBe(false);
      expect(result.error).toContain('helm not found');
    });

    it('should handle timeout errors', async () => {
      const timeoutError: any = new Error('Timeout');
      timeoutError.isTimeout = true;
      vi.mocked(execa).mockRejectedValue(timeoutError);

      const result = await checkHelmConnection();

      expect(result.working).toBe(false);
      expect(result.error).toContain('timed out');
    });
  });
});
