import React, { useEffect, useState } from 'react';
import { Box, Text } from 'ink';
import Spinner from 'ink-spinner';
import { Configuration } from './ConfigurationWizard.js';
import { saveHelmValues, installHelm, checkHelmRelease } from '../utils/helm.js';
import { join } from 'path';
import { homedir } from 'os';

interface InstallationProps {
  config: Configuration;
  onComplete: (success: boolean) => void;
}

type InstallStep = 'init' | 'values' | 'install' | 'verify' | 'complete' | 'error';

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
  ]);
  const [valuesPath, setValuesPath] = useState<string>('');

  const updateStep = (step: InstallStep, status: StepStatus['status'], error?: string) => {
    setSteps((prev) =>
      prev.map((s) => (s.step === step ? { ...s, status, error } : s))
    );
  };

  useEffect(() => {
    const runInstallation = async () => {
      try {
        // Step 1: Init
        setCurrentStep('init');
        updateStep('init', 'running');
        await new Promise((resolve) => setTimeout(resolve, 500));
        updateStep('init', 'completed');

        // Step 2: Generate Helm values
        setCurrentStep('values');
        updateStep('values', 'running');
        const outputDir = join(homedir(), '.supacontrol');
        const helmConfig = {
          jwtSecret: config.jwtSecret,
          dbPassword: config.dbPassword,
          dbHost: config.installDatabase ? undefined : config.dbHost,
          ingressEnabled: true,
          ingressHost: config.ingressHost,
          ingressClass: config.ingressClass,
          ingressDomain: config.ingressDomain,
          tlsEnabled: config.tlsEnabled,
        };
        const savedPath = await saveHelmValues(helmConfig, outputDir);
        setValuesPath(savedPath);
        updateStep('values', 'completed');

        // Step 3: Install with Helm
        setCurrentStep('install');
        updateStep('install', 'running');

        const chartPath = './charts/supacontrol';
        const result = await installHelm(
          config.namespace,
          config.releaseName,
          savedPath,
          chartPath
        );

        if (!result.success) {
          throw new Error(result.error || 'Installation failed');
        }

        updateStep('install', 'completed');

        // Step 4: Verify installation
        setCurrentStep('verify');
        updateStep('verify', 'running');
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const releaseExists = await checkHelmRelease(config.namespace, config.releaseName);
        if (!releaseExists) {
          throw new Error('Release verification failed');
        }

        updateStep('verify', 'completed');

        // Complete
        setCurrentStep('complete');
        setTimeout(() => onComplete(true), 1000);
      } catch (error: any) {
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
            <Text color={getStatusColor(step.status)}>{step.message}</Text>
            {step.error && (
              <Box marginLeft={3}>
                <Text color="red"> - {step.error}</Text>
              </Box>
            )}
          </Box>
        ))}
      </Box>

      {currentStep === 'complete' && (
        <Box marginTop={2} flexDirection="column" borderStyle="round" borderColor="green" paddingX={2} paddingY={1}>
          <Text bold color="green">
            ✓ Installation Complete!
          </Text>
          <Box marginTop={1} flexDirection="column">
            <Text>SupaControl has been installed successfully.</Text>
            <Text dimColor>Configuration saved to: {valuesPath}</Text>
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
