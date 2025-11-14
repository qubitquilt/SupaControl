import { stringify } from 'yaml';
import { writeFile, mkdir, unlink } from 'fs/promises';
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
  chartPath: string,
  dryRun: boolean = false
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
      return await upgradeHelm(namespace, releaseName, valuesPath, chartPath, dryRun);
    }

    // Validate pre-flight conditions
    console.log('Validating pre-flight conditions...');
    const helmCheck = await checkHelmConnection();
    if (!helmCheck.working) {
      throw new Error(`Helm validation failed: ${helmCheck.error}`);
    }

    console.log('Pre-flight validation passed');

    if (dryRun) {
      console.log('DRY RUN: Would perform Helm install with the following command:');
      console.log(`helm install ${releaseName} ${chartPath} --namespace ${namespace} --create-namespace --values ${valuesPath} --wait --debug`);
      console.log('DRY RUN: Installation simulation completed successfully (no changes applied).');
      return { success: true, output: 'Dry run: Helm install simulated', action: 'install' };
    }

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
  chartPath: string,
  dryRun: boolean = false
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

    if (dryRun) {
      console.log('DRY RUN: Would perform Helm upgrade with the following command:');
      console.log(`helm upgrade ${releaseName} ${chartPath} --namespace ${namespace} --values ${valuesPath} --wait --debug`);
      console.log('DRY RUN: Upgrade simulation completed successfully (no changes applied).');
      return { success: true, output: 'Dry run: Helm upgrade simulated', action: 'upgrade' };
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

export async function checkRelease(namespace: string): Promise<{exists: boolean; version: string | null}> {
  const releaseName = 'supacontrol';
  try {
    const {stdout} = await execa('helm', ['ls', '-n', namespace, '--output', 'json'], {timeout: 30000});
    const releases: any[] = JSON.parse(stdout);
    const release = releases.find((r: any) => r.name === releaseName);
    if (release) {
      // Extract version from chart name, e.g., 'supacontrol-0.1.0' -> '0.1.0'
      const chartVersion = release.chart ? release.chart.split('-').slice(1).join('-') : null;
      return {exists: true, version: chartVersion};
    }
    return {exists: false, version: null};
  } catch (error: any) {
    console.log(`No Helm releases found in namespace '${namespace}' or error: ${error.message}`);
    return {exists: false, version: null};
  }
}

export async function checkSupabaseInstance(namespace: string, instanceName: string): Promise<{exists: boolean}> {
  try {
    const {stdout} = await execa('kubectl', [
      'get',
      'supabaseinstance',
      instanceName,
      '-n',
      namespace,
      '--ignore-not-found',
      '-o',
      'json'
    ], {timeout: 30000});
    const data = JSON.parse(stdout);
    // If not found, stdout is empty object {}
    return {exists: Object.keys(data).length > 0 && data.kind === 'SupabaseInstance'};
  } catch (error: any) {
    console.log(`Error checking SupabaseInstance: ${error.message}`);
    return {exists: false};
  }
}

export function generateSupabaseInstance(config: any): string {
  const spec = {
    size: config.size || 'small',
    version: config.version || 'latest',
    components: config.components || { auth: true },
    // Add database config if internal DB
    database: config.installDatabase !== false ? {
      size: 'small', // or from config
    } : undefined,
  };
  const yamlObj = {
    apiVersion: 'supacontrol.qubitquilt.com/v1alpha1',
    kind: 'SupabaseInstance',
    metadata: {
      name: config.instanceName || 'supacontrol-instance',
      namespace: 'supacontrol-system',
    },
    spec,
  };
  return stringify(yamlObj);
}

export async function applySupabaseInstance(yamlContent: string, namespace: string, dryRun: boolean = false): Promise<{success: boolean; output?: string; error?: string}> {
  const tempPath = join('/tmp', `supabaseinstance-${Date.now()}.yaml`);
  try {
    await writeFile(tempPath, yamlContent, 'utf8');
    const args = ['apply', '-f', tempPath, '--namespace', namespace];
    if (dryRun) {
      args.push('--dry-run=client');
    }
    const {stdout} = await execa('kubectl', args, {timeout: 60000});
    if (dryRun) {
      console.log('DRY RUN: SupabaseInstance would be applied');
    } else {
      console.log('✓ SupabaseInstance applied successfully');
    }
    return {success: true, output: stdout};
  } catch (error: any) {
    const errorMsg = `Failed to apply SupabaseInstance: ${error.message}`;
    console.error(errorMsg);
    return {success: false, error: errorMsg, output: error.stdout};
  } finally {
    try {
      await unlink(tempPath);
    } catch (error) {
      const unlinkErr = error as Error;
      console.warn(`Failed to delete temp file ${tempPath}: ${unlinkErr.message}`);
    }
  }
}

export async function applySecrets(config: any, namespace: string, dryRun: boolean = false): Promise<{success: boolean; error?: string}> {
  const secretName = 'supacontrol-secrets';
  try {
    const literals: string[] = [];
    if (config.jwtSecret) literals.push(`jwt-secret=${config.jwtSecret}`);
    if (config.dbPassword) literals.push(`db-password=${config.dbPassword}`);
    // Add more literals as needed, e.g., postgresPassword from config.secrets
    if (config.secrets?.postgresPassword) literals.push(`postgres-password=${config.secrets.postgresPassword}`);
    if (literals.length === 0) {
      console.log('No secrets to apply');
      return {success: true};
    }

    if (!dryRun) {
      // Delete existing secret to force recreation/update
      try {
        await execa('kubectl', ['delete', 'secret', secretName, '-n', namespace, '--ignore-not-found=true'], {timeout: 10000});
        console.log(`Existing secret '${secretName}' deleted (if existed)`);
      } catch (deleteErr: any) {
        if (!deleteErr.message.includes('NotFound')) {
          console.warn(`Warning: Failed to delete existing secret: ${deleteErr.message}`);
        }
      }
    }

    const args = ['create', 'secret', 'generic', secretName, ...literals.map(lit => ['--from-literal', lit]).flat(), '-n', namespace];
    if (dryRun) {
      args.push('--dry-run=client', '-o', 'yaml');
      const {stdout} = await execa('kubectl', args, {timeout: 30000});
      console.log('DRY RUN: Would create/update secret');
      console.log(stdout);
    } else {
      await execa('kubectl', args, {timeout: 30000});
      console.log(`✓ Secret '${secretName}' created/updated successfully`);
    }
    return {success: true};
  } catch (error: any) {
    const errorMsg = `Failed to apply secrets: ${error.message}`;
    console.error(errorMsg);
    return {success: false, error: errorMsg};
  }
}
