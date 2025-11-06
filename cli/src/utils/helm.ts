import { stringify } from 'yaml';
import { writeFile, mkdir } from 'fs/promises';
import { join } from 'path';
import { execa } from 'execa';

export interface HelmConfig {
  jwtSecret: string;
  dbPassword: string;
  dbHost?: string;
  dbPort?: string;
  dbUser?: string;
  dbName?: string;
  ingressEnabled: boolean;
  ingressHost?: string;
  ingressClass?: string;
  ingressDomain?: string;
  tlsEnabled: boolean;
  replicas?: number;
  imageTag?: string;
}

export function generateHelmValues(config: HelmConfig): string {
  const values: any = {
    replicaCount: config.replicas || 1,
    image: {
      repository: 'supacontrol/server',
      pullPolicy: 'IfNotPresent',
      tag: config.imageTag || 'latest',
    },
    config: {
      jwtSecret: config.jwtSecret,
      database: {
        host: config.dbHost || 'supacontrol-postgresql',
        port: config.dbPort || '5432',
        user: config.dbUser || 'supacontrol',
        password: config.dbPassword,
        name: config.dbName || 'supacontrol',
      },
      kubernetes: {
        ingressClass: config.ingressClass || 'nginx',
        ingressDomain: config.ingressDomain || 'supabase.example.com',
      },
    },
    postgresql: {
      enabled: !config.dbHost || config.dbHost === 'supacontrol-postgresql',
      auth: {
        username: config.dbUser || 'supacontrol',
        password: config.dbPassword,
        database: config.dbName || 'supacontrol',
      },
      primary: {
        persistence: {
          enabled: true,
          size: '10Gi',
        },
      },
    },
  };

  if (config.ingressEnabled && config.ingressHost) {
    values.ingress = {
      enabled: true,
      className: config.ingressClass || 'nginx',
      annotations: {
        'cert-manager.io/cluster-issuer': 'letsencrypt-prod',
      },
      hosts: [
        {
          host: config.ingressHost,
          paths: [
            {
              path: '/',
              pathType: 'Prefix',
            },
          ],
        },
      ],
    };

    if (config.tlsEnabled) {
      values.ingress.tls = [
        {
          secretName: 'supacontrol-tls',
          hosts: [config.ingressHost],
        },
      ];
    }
  }

  return stringify(values);
}

export async function saveHelmValues(config: HelmConfig, outputPath: string): Promise<string> {
  const values = generateHelmValues(config);
  const filePath = join(outputPath, 'supacontrol-values.yaml');

  await mkdir(outputPath, { recursive: true });
  await writeFile(filePath, values, 'utf-8');

  return filePath;
}

export async function installHelm(
  namespace: string,
  releaseName: string,
  valuesPath: string,
  chartPath: string
): Promise<{ success: boolean; output?: string; error?: string }> {
  try {
    const { stdout } = await execa('helm', [
      'install',
      releaseName,
      chartPath,
      '--namespace',
      namespace,
      '--create-namespace',
      '--values',
      valuesPath,
      '--wait',
      '--timeout',
      '10m',
    ]);

    return { success: true, output: stdout };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function upgradeHelm(
  namespace: string,
  releaseName: string,
  valuesPath: string,
  chartPath: string
): Promise<{ success: boolean; output?: string; error?: string }> {
  try {
    const { stdout } = await execa('helm', [
      'upgrade',
      releaseName,
      chartPath,
      '--namespace',
      namespace,
      '--values',
      valuesPath,
      '--wait',
      '--timeout',
      '10m',
    ]);

    return { success: true, output: stdout };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function checkHelmRelease(namespace: string, releaseName: string): Promise<boolean> {
  try {
    await execa('helm', ['status', releaseName, '--namespace', namespace]);
    return true;
  } catch (error) {
    return false;
  }
}
