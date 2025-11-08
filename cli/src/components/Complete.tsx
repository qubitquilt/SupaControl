import React from 'react';
import { Box, Text } from 'ink';
import { Configuration } from './ConfigurationWizard.js';

interface CompleteProps {
  config: Configuration;
  success: boolean;
}

export const Complete: React.FC<CompleteProps> = ({ config, success }) => {
  if (!success) {
    return (
      <Box flexDirection="column" paddingY={1}>
        <Box marginBottom={1}>
          <Text bold color="red">
            ✗ Installation Failed
          </Text>
        </Box>
        <Box flexDirection="column">
          <Text>The installation encountered errors. Please:</Text>
          <Text>  1. Check the error messages above</Text>
          <Text>  2. Verify your Kubernetes cluster is accessible</Text>
          <Text>  3. Ensure you have proper permissions</Text>
          <Text>  4. Try running the installer again</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>For help, visit: https://github.com/qubitquilt/SupaControl</Text>
        </Box>
      </Box>
    );
  }

  const dashboardUrl = `${config.tlsEnabled ? 'https' : 'http'}://${config.ingressHost}`;

  return (
    <Box flexDirection="column" paddingY={1}>
      <Box marginBottom={1}>
        <Text bold color="green">
          🎉 Installation Successful!
        </Text>
      </Box>

      <Box marginTop={1} flexDirection="column" borderStyle="round" borderColor="green" paddingX={2} paddingY={1}>
        <Text bold>SupaControl is now running on your cluster!</Text>
      </Box>

      <Box marginTop={1} flexDirection="column">
        <Box marginTop={1}>
          <Text bold color="cyan">
            📍 Access Information:
          </Text>
        </Box>
        <Box marginTop={1} flexDirection="column" paddingX={2}>
          <Text>
            <Text color="gray">Dashboard URL: </Text>
            <Text bold color="cyan">
              {dashboardUrl}
            </Text>
          </Text>
          <Text>
            <Text color="gray">Namespace: </Text>
            <Text bold>{config.namespace}</Text>
          </Text>
          <Text>
            <Text color="gray">Release: </Text>
            <Text bold>{config.releaseName}</Text>
          </Text>
        </Box>
      </Box>

      <Box marginTop={1} flexDirection="column">
        <Box marginTop={1}>
          <Text bold color="yellow">
            🔐 Default Credentials:
          </Text>
        </Box>
        <Box marginTop={1} flexDirection="column" paddingX={2}>
          <Text>
            <Text color="gray">Username: </Text>
            <Text bold>admin</Text>
          </Text>
          <Text>
            <Text color="gray">Password: </Text>
            <Text bold>admin</Text>
          </Text>
          <Box marginTop={1}>
            <Text color="red" bold>
              ⚠️  IMPORTANT: Change the default password immediately after first login!
            </Text>
          </Box>
        </Box>
      </Box>

      <Box marginTop={1} flexDirection="column">
        <Box marginTop={1}>
          <Text bold color="cyan">
            🚀 Next Steps:
          </Text>
        </Box>
        <Box marginTop={1} flexDirection="column" paddingX={2}>
          <Text>1. Wait for all pods to be ready:</Text>
          <Text dimColor>   kubectl get pods -n {config.namespace} --watch</Text>
          <Box marginTop={1}>
            <Text>2. Access the dashboard:</Text>
          </Box>
          <Text dimColor>   {dashboardUrl}</Text>
          <Box marginTop={1}>
            <Text>3. Login with default credentials and change password</Text>
          </Box>
          <Box marginTop={1}>
            <Text>4. Generate an API key in Settings for CLI access</Text>
          </Box>
          <Box marginTop={1}>
            <Text>5. Create your first Supabase instance!</Text>
          </Box>
        </Box>
      </Box>

      {config.tlsEnabled && (
        <Box marginTop={1} flexDirection="column" borderStyle="round" borderColor="yellow" paddingX={2} paddingY={1}>
          <Text color="yellow" bold>
            📝 Note about TLS/HTTPS:
          </Text>
          <Text>
            TLS is enabled with cert-manager. Make sure cert-manager is installed and configured in your cluster.
          </Text>
          <Box marginTop={1}>
            <Text dimColor >
              If cert-manager is not installed, the dashboard may not be accessible via HTTPS immediately.
            </Text>
          </Box>
        </Box>
      )}

      <Box marginTop={1} flexDirection="column">
        <Box marginTop={1}>
          <Text bold>Useful Commands:</Text>
        </Box>
        <Box marginTop={1} flexDirection="column" paddingX={2}>
          <Text dimColor>• View logs: kubectl logs -n {config.namespace} -l app.kubernetes.io/name=supacontrol -f</Text>
          <Text dimColor>• Check status: helm status {config.releaseName} -n {config.namespace}</Text>
          <Text dimColor>• Port forward: kubectl port-forward -n {config.namespace} svc/{config.releaseName} 8080:8080</Text>
        </Box>
      </Box>

      <Box marginTop={2} borderStyle="round" borderColor="cyan" paddingX={2} paddingY={1}>
        <Text>
          📖 Documentation: <Text color="cyan">https://github.com/qubitquilt/SupaControl</Text>
        </Text>
      </Box>
    </Box>
  );
};
