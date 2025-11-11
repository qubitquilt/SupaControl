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
  certManagerIssuer?: string;
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
      annotations: config.tlsEnabled && config.certManagerIssuer ? {
        'cert-manager.io/cluster-issuer': config.certManagerIssuer,
      } : {},
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
): Promise<{ success: boolean; output?: string; error?: string; action?: 'install' | 'upgrade' | 'cleanup' }> {
  try {
    console.log(`Starting installation of '${releaseName}' in namespace '${namespace}'...`);
    console.log(`Chart path: ${chartPath}`);
    console.log(`Values file: ${valuesPath}`);
    
    // First check if release already exists
    const existingRelease = await checkHelmRelease(namespace, releaseName);
    
    if (existingRelease) {
      // If release exists, try to upgrade instead
      console.log(`Release '${releaseName}' already exists. Attempting upgrade...`);
      return await upgradeHelm(namespace, releaseName, valuesPath, chartPath);
    }
    
    // Validate pre-flight conditions
    console.log('Validating pre-flight conditions...');
    const helmCheck = await checkHelmConnection();
    if (!helmCheck.working) {
      throw new Error(`Helm validation failed: ${helmCheck.error}`);
    }
    
    console.log('Pre-flight validation passed');
    
    // Do a dry-run first to validate the installation
    console.log('Performing dry-run validation...');
    try {
      await runHelmCommand([
        'install',
        releaseName,
        chartPath,
        '--namespace',
        namespace,
        '--create-namespace',
        '--values',
        valuesPath,
        '--dry-run',
        '--debug',
      ], {}, 30000, 'Helm install dry-run');
      console.log('✓ Dry-run validation successful');
    } catch (dryRunError: any) {
      console.log('Dry-run failed:', dryRunError.message);
      // Continue anyway, dry-run can fail for various reasons
    }
    
    console.log('Starting actual installation...');
    
    // Now do the actual installation
    const { stdout } = await runHelmCommand([
      'install',
      releaseName,
      chartPath,
      '--namespace',
      namespace,
      '--create-namespace',
      '--values',
      valuesPath,
      '--wait',
      '--debug',
    ], {}, 120000, 'Helm install'); // 2 minute total timeout

    console.log('✓ Installation completed successfully');
    return { success: true, output: stdout, action: 'install' };
  } catch (error: any) {
    console.error('Helm install error:', error.message);
    
    // Enhanced error handling
    let errorMessage = error.message;
    
    if (error.message?.includes('Command timeout')) {
      errorMessage = `Helm install timed out after 2 minutes. This typically indicates: 1) Slow cluster response, 2) Network connectivity issues, 3) Resource constraints, or 4) Image pull issues. Check: kubectl get pods -n ${namespace} && kubectl get events -n ${namespace}`;
    } else if (error.isTimeout) {
      errorMessage = 'Installation timed out. Check cluster status and network connectivity.';
    } else if (error.message?.includes('timed out')) {
      errorMessage = 'Installation timed out. This may be due to network issues or resource constraints.';
    } else if (error.message?.includes('context deadline exceeded')) {
      errorMessage = 'Installation exceeded time limit. Check cluster resources and network connectivity.';
    } else if (error.message?.includes('failed to download')) {
      errorMessage = 'Failed to download chart. Check your internet connection and Helm repository configuration.';
    } else if (error.message?.includes('no space left on device')) {
      errorMessage = 'Insufficient disk space. Free up some space and try again.';
    } else if (error.message?.includes('permission denied')) {
      errorMessage = 'Permission denied. Ensure you have proper cluster access and permissions.';
    } else if (error.message?.includes('connection refused')) {
      errorMessage = 'Unable to connect to Kubernetes cluster. Check your kubeconfig and cluster status.';
    } else if (error.message?.includes('cannot re-use a name that is still in use')) {
      errorMessage = 'A release with this name already exists in the namespace. The installer will attempt to upgrade the existing release.';
    } else if (error.message?.includes('could not get apiVersions from Kubernetes')) {
      errorMessage = 'Cannot connect to Kubernetes cluster. Verify kubectl cluster-info shows your cluster.';
    }
    
    return { success: false, error: errorMessage, output: error.stdout };
  }
}

async function runHelmCommand(
  args: string[],
  options: any,
  timeoutMs: number,
  description: string
): Promise<{ stdout: string; stderr: string }> {
  try {
    console.log(`Executing: helm ${args.join(' ')}`);
    const result = await execa('helm', args, {
      ...options,
      timeout: timeoutMs,
    });
    return { stdout: result.stdout, stderr: result.stderr };
  } catch (error: any) {
    if (error.isTimeout) {
      throw new Error(`Command timeout: ${description} took longer than ${timeoutMs}ms`);
    }
    throw error;
  }
}

export async function upgradeHelm(
  namespace: string,
  releaseName: string,
  valuesPath: string,
  chartPath: string
): Promise<{ success: boolean; output?: string; error?: string; action?: 'upgrade' }> {
  try {
    console.log(`Attempting to upgrade release '${releaseName}' in namespace '${namespace}'...`);
    console.log(`Chart path: ${chartPath}`);
    console.log(`Values file: ${valuesPath}`);
    
    // First, try a dry-run to validate the upgrade without actually executing it
    console.log('Performing dry-run validation...');
    try {
      await runHelmCommand([
        'upgrade',
        releaseName,
        chartPath,
        '--namespace',
        namespace,
        '--values',
        valuesPath,
        '--dry-run',
        '--debug',
      ], {}, 30000, 'Helm dry-run');
      console.log('✓ Dry-run validation successful');
    } catch (dryRunError: any) {
      console.log('Dry-run failed:', dryRunError.message);
      // Continue anyway, dry-run can fail for various reasons
    }
    
    console.log('Starting actual upgrade...');
    
    // Now do the actual upgrade with a shorter timeout and better error handling
    const { stdout } = await runHelmCommand([
      'upgrade',
      releaseName,
      chartPath,
      '--namespace',
      namespace,
      '--values',
      valuesPath,
      '--wait',
      '--debug',
    ], {}, 70000, 'Helm upgrade'); // 70 second total timeout

    console.log('✓ Upgrade completed successfully');
    return { success: true, output: stdout, action: 'upgrade' };
  } catch (error: any) {
    console.error('Helm upgrade error:', error.message);
    
    // Enhanced error handling with specific timeout detection
    let errorMessage = error.message;
    
    if (error.message?.includes('Command timeout')) {
      errorMessage = `Helm upgrade timed out after 70 seconds. This typically indicates: 1) Slow cluster response, 2) Network connectivity issues, 3) Resource constraints, or 4) Stuck pods. Try: kubectl get pods -n ${namespace} && kubectl get events -n ${namespace}`;
    } else if (error.isTimeout || error.message?.includes('timed out')) {
      errorMessage = 'Helm upgrade timed out. Check cluster status and network connectivity.';
    } else if (error.message?.includes('context deadline exceeded')) {
      errorMessage = 'Helm upgrade exceeded time limit. Check cluster resources and network connectivity.';
    } else if (error.message?.includes('no space left on device')) {
      errorMessage = 'Insufficient disk space. Free up some space and try again.';
    } else if (error.message?.includes('permission denied')) {
      errorMessage = 'Permission denied. Ensure you have proper cluster access and permissions.';
    } else if (error.message?.includes('connection refused')) {
      errorMessage = 'Unable to connect to Kubernetes cluster. Check your kubeconfig and cluster status.';
    } else if (error.message?.includes('could not get apiVersions from Kubernetes')) {
      errorMessage = 'Cannot connect to Kubernetes cluster. Verify kubectl cluster-info shows your cluster.';
    }
    
    return { success: false, error: errorMessage, output: error.stdout };
  }
}

export async function uninstallHelm(
  namespace: string,
  releaseName: string
): Promise<{ success: boolean; output?: string; error?: string; action?: 'uninstall' }> {
  try {
    const { stdout } = await execa('helm', [
      'uninstall',
      releaseName,
      '--namespace',
      namespace,
    ]);

    return { success: true, output: stdout, action: 'uninstall' };
  } catch (error: any) {
    return { success: false, error: error.message, output: error.stdout };
  }
}

export async function checkHelmRelease(namespace: string, releaseName: string): Promise<boolean> {
  try {
    const { stdout } = await execa('helm', ['status', releaseName, '--namespace', namespace, '--output', 'json'], {
      timeout: 30000, // 30 second timeout
    });
    return typeof stdout === 'string' && stdout.length > 0;
  } catch (error) {
    console.log(`Release '${releaseName}' not found in namespace '${namespace}'`);
    return false;
  }
}

export interface PodStatus {
  name: string;
  status: string;
  ready: string;
  restarts: number;
  age: string;
  imagePullErrors?: string[];
}

export async function checkPodStatus(namespace: string, releaseName: string): Promise<{ success: boolean; pods: PodStatus[]; errors: string[] }> {
  try {
    // Get pod information
    const { stdout: podsOutput } = await execa('kubectl', [
      'get', 'pods',
      '-l', `app.kubernetes.io/instance=${releaseName}`,
      '--namespace', namespace,
      '--output', 'json'
    ], {
      timeout: 30000,
    });

    const podData = JSON.parse(podsOutput);
    const pods: PodStatus[] = [];
    const errors: string[] = [];

    for (const item of podData.items || []) {
      const podStatus: PodStatus = {
        name: item.metadata.name,
        status: item.status.phase,
        ready: `${item.status.containerStatuses?.filter((cs: any) => cs.ready).length || 0}/${item.spec.containers?.length || 0}`,
        restarts: item.status.containerStatuses?.reduce((sum: number, cs: any) => sum + (cs.restartCount || 0), 0) || 0,
        age: item.status.startTime ? new Date(item.status.startTime).toLocaleString() : 'Unknown',
      };

      // Check for image pull errors
      const containerStatuses = item.status.containerStatuses || [];
      const imagePullErrors: string[] = [];
      
      for (const cs of containerStatuses) {
        if (cs.state?.waiting?.reason === 'ErrImagePull' || cs.state?.waiting?.reason === 'ImagePullBackOff') {
          imagePullErrors.push(`Container ${cs.name}: ${cs.state.waiting.message || cs.state.waiting.reason}`);
        }
        if (cs.state?.waiting?.reason === 'CrashLoopBackOff' && cs.lastState?.terminated?.exitCode !== 0) {
          imagePullErrors.push(`Container ${cs.name}: Crashed with exit code ${cs.lastState.terminated.exitCode}`);
        }
      }

      if (imagePullErrors.length > 0) {
        podStatus.imagePullErrors = imagePullErrors;
        errors.push(`Pod ${podStatus.name} has issues: ${imagePullErrors.join(', ')}`);
      }

      pods.push(podStatus);
    }

    // Check if all pods are running successfully
    const allPodsRunning = pods.length > 0 && pods.every(pod => 
      pod.status === 'Running' && !pod.imagePullErrors?.length
    );

    return { 
      success: allPodsRunning, 
      pods, 
      errors: errors.length > 0 ? errors : [] 
    };
  } catch (error: any) {
    return { 
      success: false, 
      pods: [], 
      errors: [`Failed to check pod status: ${error.message}`] 
    };
  }
}

// Helper function to check if Helm is working properly
export async function checkHelmConnection(): Promise<{ working: boolean; error?: string }> {
  try {
    const { stdout } = await execa('helm', ['version', '--short'], {
      timeout: 10000, // 10 second timeout
    });
    console.log('Helm version:', stdout);
    return { working: true };
  } catch (error: any) {
    const errorMessage = error.isTimeout ? 'Helm command timed out' : error.message;
    console.error('Helm connection check failed:', errorMessage);
    return { working: false, error: errorMessage };
  }
}
