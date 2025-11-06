import { execa } from 'execa';

export interface PrerequisiteCheck {
  name: string;
  command: string;
  args?: string[];
  versionCommand?: string[];
  minVersion?: string;
  required: boolean;
  installUrl?: string;
}

export interface PrerequisiteResult {
  name: string;
  installed: boolean;
  version?: string;
  required: boolean;
  installUrl?: string;
}

const prerequisites: PrerequisiteCheck[] = [
  {
    name: 'kubectl',
    command: 'kubectl',
    args: ['version', '--client', '--short'],
    required: true,
    installUrl: 'https://kubernetes.io/docs/tasks/tools/',
  },
  {
    name: 'helm',
    command: 'helm',
    args: ['version', '--short'],
    minVersion: '3.0.0',
    required: true,
    installUrl: 'https://helm.sh/docs/intro/install/',
  },
  {
    name: 'docker',
    command: 'docker',
    args: ['--version'],
    required: false,
    installUrl: 'https://docs.docker.com/get-docker/',
  },
  {
    name: 'git',
    command: 'git',
    args: ['--version'],
    required: false,
    installUrl: 'https://git-scm.com/downloads',
  },
];

async function checkCommand(check: PrerequisiteCheck): Promise<PrerequisiteResult> {
  try {
    const result = await execa(check.command, check.args || ['--version']);
    return {
      name: check.name,
      installed: true,
      version: result.stdout.trim(),
      required: check.required,
      installUrl: check.installUrl,
    };
  } catch (error) {
    return {
      name: check.name,
      installed: false,
      required: check.required,
      installUrl: check.installUrl,
    };
  }
}

export async function checkPrerequisites(): Promise<PrerequisiteResult[]> {
  const results = await Promise.all(prerequisites.map(checkCommand));
  return results;
}

export async function checkKubernetesConnection(): Promise<boolean> {
  try {
    await execa('kubectl', ['cluster-info']);
    return true;
  } catch (error) {
    return false;
  }
}

export async function getKubernetesNamespaces(): Promise<string[]> {
  try {
    const result = await execa('kubectl', ['get', 'namespaces', '-o', 'jsonpath={.items[*].metadata.name}']);
    return result.stdout.split(' ').filter(Boolean);
  } catch (error) {
    return [];
  }
}
