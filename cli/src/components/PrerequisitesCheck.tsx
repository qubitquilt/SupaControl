import React, { useEffect, useState } from 'react';
import { Box, Text } from 'ink';
import Spinner from 'ink-spinner';
import { checkPrerequisites, checkKubernetesConnection, PrerequisiteResult } from '../utils/prerequisites.js';

interface PrerequisitesCheckProps {
  onComplete: (success: boolean) => void;
}

export const PrerequisitesCheck: React.FC<PrerequisitesCheckProps> = ({ onComplete }) => {
  const [checking, setChecking] = useState(true);
  const [results, setResults] = useState<PrerequisiteResult[]>([]);
  const [k8sConnected, setK8sConnected] = useState<boolean | null>(null);

  useEffect(() => {
    const runChecks = async () => {
      // Check prerequisites
      const prereqResults = await checkPrerequisites();
      setResults(prereqResults);

      // Check Kubernetes connection
      const k8sOk = await checkKubernetesConnection();
      setK8sConnected(k8sOk);

      setChecking(false);

      // Determine if all required checks passed
      const requiredMissing = prereqResults.filter(r => r.required && !r.installed);
      const success = requiredMissing.length === 0 && k8sOk;

      setTimeout(() => onComplete(success), 1000);
    };

    runChecks();
  }, [onComplete]);

  const getStatusIcon = (installed: boolean) => {
    return installed ? '✓' : '✗';
  };

  const getStatusColor = (installed: boolean) => {
    return installed ? 'green' : 'red';
  };

  return (
    <Box flexDirection="column" paddingY={1}>
      <Box marginBottom={1}>
        <Text bold color="cyan">
          📋 Checking Prerequisites
        </Text>
      </Box>

      {checking && (
        <Box>
          <Text color="cyan">
            <Spinner type="dots" />
          </Text>
          <Text> Checking system requirements...</Text>
        </Box>
      )}

      {!checking && (
        <Box flexDirection="column">
          {results.map((result) => (
            <Box key={result.name} marginY={0}>
              <Text color={getStatusColor(result.installed)} bold>
                {getStatusIcon(result.installed)}
              </Text>
              <Text> {result.name}: </Text>
              {result.installed ? (
                <Text color="green">{result.version || 'Installed'}</Text>
              ) : (
                <Box>
                  <Text color="red">Not found</Text>
                  {result.required && (
                    <Text color="yellow"> (Required - Install: {result.installUrl})</Text>
                  )}
                </Box>
              )}
            </Box>
          ))}

          <Box marginTop={1}>
            <Text color={getStatusColor(k8sConnected || false)} bold>
              {getStatusIcon(k8sConnected || false)}
            </Text>
            <Text> Kubernetes Cluster Connection: </Text>
            {k8sConnected ? (
              <Text color="green">Connected</Text>
            ) : (
              <Text color="red">Not connected (Check kubectl configuration)</Text>
            )}
          </Box>

          <Box marginTop={1} borderStyle="round" borderColor="gray" paddingX={2} paddingY={1}>
            {results.filter(r => r.required && !r.installed).length === 0 && k8sConnected ? (
              <Text color="green" bold>
                ✓ All prerequisites met! Ready to proceed.
              </Text>
            ) : (
              <Box flexDirection="column">
                <Text color="red" bold>
                  ✗ Some prerequisites are missing
                </Text>
                <Text>Please install the required tools and ensure Kubernetes is accessible.</Text>
              </Box>
            )}
          </Box>
        </Box>
      )}
    </Box>
  );
};
