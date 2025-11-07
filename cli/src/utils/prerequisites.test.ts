import { describe, it, expect, vi, beforeEach } from 'vitest';
import { execa } from 'execa';
import {
  checkPrerequisites,
  checkKubernetesConnection,
  getKubernetesNamespaces,
} from './prerequisites.js';

// Mock execa
vi.mock('execa');

describe('prerequisites utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('checkPrerequisites', () => {
    it('should return all prerequisites when all are installed', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'v1.28.0',
        stderr: '',
        exitCode: 0,
      } as any);

      const results = await checkPrerequisites();

      expect(results).toHaveLength(4);
      expect(results.every(r => r.installed)).toBe(true);
      expect(results.find(r => r.name === 'kubectl')).toBeDefined();
      expect(results.find(r => r.name === 'helm')).toBeDefined();
      expect(results.find(r => r.name === 'docker')).toBeDefined();
      expect(results.find(r => r.name === 'git')).toBeDefined();
    });

    it('should correctly identify missing prerequisites', async () => {
      (vi.mocked(execa).mockImplementation as any)(async (file: string, args?: string[] | any) => {
        if (file === 'kubectl' || file === 'helm') {
          return {
            stdout: 'v1.28.0',
            stderr: '',
            exitCode: 0,
          };
        }
        throw new Error('Command not found');
      });

      const results = await checkPrerequisites();

      const kubectl = results.find(r => r.name === 'kubectl');
      const docker = results.find(r => r.name === 'docker');

      expect(kubectl?.installed).toBe(true);
      expect(docker?.installed).toBe(false);
    });

    it('should mark kubectl and helm as required', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Not found'));

      const results = await checkPrerequisites();

      const kubectl = results.find(r => r.name === 'kubectl');
      const helm = results.find(r => r.name === 'helm');
      const docker = results.find(r => r.name === 'docker');

      expect(kubectl?.required).toBe(true);
      expect(helm?.required).toBe(true);
      expect(docker?.required).toBe(false);
    });

    it('should include install URLs for missing prerequisites', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Not found'));

      const results = await checkPrerequisites();

      expect(results.every(r => r.installUrl)).toBe(true);
      expect(results.find(r => r.name === 'kubectl')?.installUrl).toContain('kubernetes.io');
      expect(results.find(r => r.name === 'helm')?.installUrl).toContain('helm.sh');
    });

    it('should capture version information', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'Client Version: v1.28.0',
        stderr: '',
        exitCode: 0,
      } as any);

      const results = await checkPrerequisites();

      expect(results.every(r => r.version)).toBe(true);
      expect(results[0].version).toContain('1.28.0');
    });
  });

  describe('checkKubernetesConnection', () => {
    it('should return true when kubectl cluster-info succeeds', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'Kubernetes control plane is running at https://...',
        stderr: '',
        exitCode: 0,
      } as any);

      const result = await checkKubernetesConnection();

      expect(result).toBe(true);
      expect(execa).toHaveBeenCalledWith('kubectl', ['cluster-info']);
    });

    it('should return false when kubectl cluster-info fails', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Connection refused'));

      const result = await checkKubernetesConnection();

      expect(result).toBe(false);
    });
  });

  describe('getKubernetesNamespaces', () => {
    it('should return an array of namespace names', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'default kube-system kube-public kube-node-lease',
        stderr: '',
        exitCode: 0,
      } as any);

      const namespaces = await getKubernetesNamespaces();

      expect(namespaces).toEqual([
        'default',
        'kube-system',
        'kube-public',
        'kube-node-lease',
      ]);
      expect(execa).toHaveBeenCalledWith('kubectl', [
        'get',
        'namespaces',
        '-o',
        'jsonpath={.items[*].metadata.name}',
      ]);
    });

    it('should return an empty array when kubectl fails', async () => {
      vi.mocked(execa).mockRejectedValue(new Error('Connection refused'));

      const namespaces = await getKubernetesNamespaces();

      expect(namespaces).toEqual([]);
    });

    it('should filter out empty strings', async () => {
      vi.mocked(execa).mockResolvedValue({
        stdout: 'default  kube-system  ',
        stderr: '',
        exitCode: 0,
      } as any);

      const namespaces = await getKubernetesNamespaces();

      expect(namespaces).toEqual(['default', 'kube-system']);
      expect(namespaces).not.toContain('');
    });
  });
});
