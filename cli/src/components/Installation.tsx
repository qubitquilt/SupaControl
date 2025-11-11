import React, { useEffect, useState } from 'react';
import { Box, Text } from 'ink';
import Spinner from 'ink-spinner';
import { Configuration } from './ConfigurationWizard.js';
import { saveHelmValues, installHelm, checkHelmRelease, checkPodStatus } from '../utils/helm.js';
import { join, resolve } from 'path';
import { homedir } from 'os';
import { access, constants } from 'fs/promises';
import { execa } from 'execa';

interface InstallationProps {
  config: Configuration;
  onComplete: (success: boolean) => void;
}

type InstallStep = 'init' | 'values' | 'install' | 'verify' | 'pods' | 'complete' | 'error';

interface StepStatus {
  step: InstallStep;
  message: string;
  status: 'pending' | 'running' | 'completed' | 'error';
  error?: string;
}

export const Installation: React.FC<InstallationProps> = ({ config, onComplete }) => {
  const [currentStep, setCurrentStep] = useState<InstallStep>('init');
  const [steps, setSteps] = useState<StepStatus[]>([
    { step: 'init', message: 'Initializing installation', status: 'pending' },
    { step: 'values', message: 'Generating Helm values', status: 'pending' },
    { step: 'install', message: 'Installing SupaControl', status: 'pending' },
    { step: 'verify', message: 'Verifying installation', status: 'pending' },
    { step: 'pods', message: 'Checking pod health', status: 'pending' },
  ]);
  const [valuesPath, setValuesPath] = useState<string>('');
  const [chartPath, setChartPath] = useState<string>('');
  const [installAction, setInstallAction] = useState<'install' | 'upgrade'>('install');

  const updateStep = (step: InstallStep, status: StepStatus['status'], error?: string) => {
    setSteps((prev) =>
      prev.map((s) => {
        if (s.step === step) {
          if (step === 'install' && (status === 'completed' || status === 'running')) {
            return {
              ...s,
              status,
              error,
              message: installAction === 'upgrade' ? 'Upgrading SupaControl' : 'Installing SupaControl'
            };
          }
          return { ...s, status, error };
        }
        return s;
      })
    );
  };

  // Determine the chart path
  const getChartPath = async (): Promise<string> => {
    // First, check environment variable
    const envChartPath = process.env.SUPACONTROL_CHART_PATH;
    if (envChartPath) {
      try {
        await access(envChartPath, constants.R_OK);
        return envChartPath;
      } catch {
        // Environment path doesn't exist, continue to other options
      }
    }

    // Check if running from repository root
    const localChartPath = resolve(process.cwd(), 'charts/supacontrol');
    try {
      await access(localChartPath, constants.R_OK);
      console.log(`Found chart at: ${localChartPath}`);
      return localChartPath;
    } catch {
      // Local path doesn't exist
    }

    // Check common parent directory (if CLI is in cli/ subdirectory)
    const parentChartPath = resolve(process.cwd(), '../charts/supacontrol');
    try {
      await access(parentChartPath, constants.R_OK);
      console.log(`Found chart at: ${parentChartPath}`);
      return parentChartPath;
    } catch {
      // Parent path doesn't exist
    }

    // Try to find chart in common locations
    const commonPaths = [
      resolve('/usr/local/share/supacontrol/charts/supacontrol'),
      resolve('/opt/supacontrol/charts/supacontrol'),
      resolve(join(homedir(), 'supacontrol/charts/supacontrol')),
    ];

    for (const path of commonPaths) {
      try {
        await access(path, constants.R_OK);
        console.log(`Found chart at: ${path}`);
        return path;
      } catch {
        // Path doesn't exist
      }
    }

    // Clone repository to temporary directory with timeout
    const tmpDir = join(homedir(), '.supacontrol', 'repo');
    try {
      console.log('Attempting to clone SupaControl repository...');
      
      // Add timeout to git clone and use shallow clone for speed
      const subprocess = execa('git', [
        'clone',
        '--depth', '1',
        '--branch', 'main',
        '--single-branch',
        'https://github.com/qubitquilt/SupaControl.git',
        tmpDir
      ], {
        timeout: 30000, // 30 second timeout
        maxBuffer: 1024 * 1024, // 1MB buffer
      });
      
      await subprocess;
      const chartPath = join(tmpDir, 'charts/supacontrol');
      console.log(`Cloned chart to: ${chartPath}`);
      return chartPath;
    } catch (error: any) {
      if (error.isTimeout) {
        throw new Error('Repository clone timed out. Check your internet connection or set SUPACONTROL_CHART_PATH environment variable.');
      }
      throw new Error(`Failed to locate or clone Helm chart. Please set SUPACONTROL_CHART_PATH environment variable or run from the repository root. Error: ${error.message}`);
    }
  };

  useEffect(() => {
    const runInstallation = async () => {
      // Add overall timeout to prevent indefinite hanging
      const installationTimeout = setTimeout(() => {
        updateStep(currentStep, 'error', 'Installation process timed out. This may be due to network issues, resource constraints, or cluster connectivity problems.');
        setCurrentStep('error');
        setTimeout(() => onComplete(false), 3000);
      }, 300000); // 5 minute overall timeout

      try {
        console.log('Starting SupaControl installation process...');
        
        // Step 1: Init
        setCurrentStep('init');
        updateStep('init', 'running');
        await new Promise((resolve) => setTimeout(resolve, 500));
        updateStep('init', 'completed');
        console.log('✓ Initialization complete');

        // Step 2: Generate Helm values
        setCurrentStep('values');
        updateStep('values', 'running');
        const outputDir = join(homedir(), '.supacontrol');
        console.log('Generating Helm values...');

        // Determine chart path
        const resolvedChartPath = await getChartPath();
        setChartPath(resolvedChartPath);

        const helmConfig = {
          jwtSecret: config.jwtSecret,
          dbPassword: config.dbPassword,
          dbHost: config.installDatabase ? undefined : config.dbHost,
          dbPort: config.installDatabase ? undefined : config.dbPort,
          dbUser: config.installDatabase ? undefined : config.dbUser,
          dbName: config.installDatabase ? undefined : config.dbName,
          ingressEnabled: true,
          ingressHost: config.ingressHost,
          ingressClass: config.ingressClass,
          ingressDomain: config.ingressDomain,
          tlsEnabled: config.tlsEnabled,
          certManagerIssuer: config.certManagerIssuer,
        };
        const savedPath = await saveHelmValues(helmConfig, outputDir);
        setValuesPath(savedPath);
        updateStep('values', 'completed');
        console.log('✓ Helm values generated and saved');

        // Step 3: Install with Helm
        setCurrentStep('install');
        updateStep('install', 'running');
        console.log('Starting Helm installation/upgrade...');

        const result = await installHelm(
          config.namespace,
          config.releaseName,
          savedPath,
          resolvedChartPath
        );

        if (!result.success) {
          throw new Error(result.error || 'Installation failed');
        }

        // Store the action taken and update the step message directly
        if (result.action === 'upgrade') {
          setInstallAction('upgrade');
          setSteps(prev => prev.map(s =>
            s.step === 'install'
              ? { ...s, message: 'Upgrading SupaControl', status: 'completed' }
              : s
          ));
          console.log('✓ Upgrade completed successfully');
        } else {
          setInstallAction('install');
          setSteps(prev => prev.map(s =>
            s.step === 'install'
              ? { ...s, message: 'Installing SupaControl', status: 'completed' }
              : s
          ));
          console.log('✓ Installation completed successfully');
        }

        // Step 4: Verify installation
        setCurrentStep('verify');
        updateStep('verify', 'running');
        console.log('Verifying installation...');
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const releaseExists = await checkHelmRelease(config.namespace, config.releaseName);
        if (!releaseExists) {
          throw new Error('Release verification failed - Helm release not found after installation');
        }

        updateStep('verify', 'completed');
        console.log('✓ Installation verified successfully');

        // Step 5: Check pod health
        setCurrentStep('pods');
        updateStep('pods', 'running');
        console.log('Checking pod health...');
        
        // Wait a bit more for pods to start
        await new Promise((resolve) => setTimeout(resolve, 3000));
        
        const podStatus = await checkPodStatus(config.namespace, config.releaseName);
        
        if (!podStatus.success) {
          let errorMessage = 'Pod health check failed. ';
          
          if (podStatus.errors.length > 0) {
            errorMessage += `Issues found: ${podStatus.errors.join('; ')}. `;
          }
          
          if (podStatus.pods.length === 0) {
            errorMessage += 'No pods found for the release.';
          } else {
            const podInfo = podStatus.pods.map(pod =>
              `${pod.name}: ${pod.status}${pod.imagePullErrors ? ' (ERRORS: ' + pod.imagePullErrors.join(', ') + ')' : ''}`
            ).join('; ');
            errorMessage += `Pod status: ${podInfo}`;
          }
          
          errorMessage += '\n\nTroubleshooting suggestions:\n';
          errorMessage += '1. Check image availability: kubectl describe pod <pod-name> -n ' + config.namespace + '\n';
          errorMessage += '2. Check events: kubectl get events -n ' + config.namespace + ' --sort-by=\'.lastTimestamp\'\n';
          errorMessage += '3. Verify image pull policy and registry access\n';
          errorMessage += '4. Check cluster resources and network connectivity';
          
          throw new Error(errorMessage);
        }

        updateStep('pods', 'completed');
        console.log('✓ Pod health check successful');

        // Clear timeout and complete
        clearTimeout(installationTimeout);
        setCurrentStep('complete');
        setTimeout(() => onComplete(true), 1000);
        
      } catch (error: any) {
        clearTimeout(installationTimeout);
        console.error('Installation failed:', error.message);
        updateStep(currentStep, 'error', error.message);
        setCurrentStep('error');
        setTimeout(() => onComplete(false), 2000);
      }
    };

    runInstallation();
  }, []);

  const getStatusIcon = (status: StepStatus['status']) => {
    switch (status) {
      case 'running':
        return <Spinner type="dots" />;
      case 'completed':
        return <Text color="green">✓</Text>;
      case 'error':
        return <Text color="red">✗</Text>;
      default:
        return <Text color="gray">◯</Text>;
    }
  };

  const getStatusColor = (status: StepStatus['status']) => {
    switch (status) {
      case 'running':
        return 'cyan';
      case 'completed':
        return 'green';
      case 'error':
        return 'red';
      default:
        return 'gray';
    }
  };

  return (
    <Box flexDirection="column" paddingY={1}>
      <Box marginBottom={1}>
        <Text bold color="cyan">
          🚀 Installing SupaControl
        </Text>
      </Box>

      <Box flexDirection="column">
        {steps.map((step) => (
          <Box key={step.step} marginY={0}>
            <Box width={3}>{getStatusIcon(step.status)}</Box>
            <Box flexDirection="column" width="100%">
              <Text color={getStatusColor(step.status)}>{step.message}</Text>
              {step.error && (
                <Box marginLeft={0} marginTop={1} flexDirection="column">
                  <Text color="red" wrap="wrap">{step.error}</Text>
                </Box>
              )}
            </Box>
          </Box>
        ))}
      </Box>

      {currentStep === 'complete' && (
        <Box marginTop={2} flexDirection="column" borderStyle="round" borderColor="green" paddingX={2} paddingY={1}>
          <Text bold color="green">
            ✓ {installAction === 'upgrade' ? 'Upgrade' : 'Installation'} Complete!
          </Text>
          <Box marginTop={1} flexDirection="column">
            <Text>SupaControl has been {installAction === 'upgrade' ? 'upgraded' : 'installed'} successfully.</Text>
            <Text dimColor>Configuration saved to: {valuesPath}</Text>
            {installAction === 'upgrade' && (
              <Text dimColor>Existing release was updated with new configuration.</Text>
            )}
          </Box>
        </Box>
      )}

      {currentStep === 'error' && (
        <Box marginTop={2} flexDirection="column" borderStyle="round" borderColor="red" paddingX={2} paddingY={1}>
          <Text bold color="red">
            ✗ Installation Failed
          </Text>
          <Text>Please check the error messages above and try again.</Text>
        </Box>
      )}
    </Box>
  );
};
