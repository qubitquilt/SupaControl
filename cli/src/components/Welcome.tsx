import React from 'react';
import { Box, Text } from 'ink';
import BigText from 'ink-big-text';
import Gradient from 'ink-gradient';

export const Welcome: React.FC = () => {
  return (
    <Box flexDirection="column" paddingY={1}>
      <Gradient name="rainbow">
        <BigText text="SupaControl" font="tiny" />
      </Gradient>
      <Box marginTop={1} flexDirection="column">
        <Text bold color="cyan">
          🚀 Supabase Management Platform Installer
        </Text>
        <Text dimColor>
          This wizard will guide you through installing SupaControl on your Kubernetes cluster.
        </Text>
      </Box>
      <Box marginTop={1} flexDirection="column" borderStyle="round" borderColor="gray" paddingX={2} paddingY={1}>
        <Text bold color="yellow">
          What will be installed:
        </Text>
        <Text>  • SupaControl API Server</Text>
        <Text>  • PostgreSQL Database (for instance inventory)</Text>
        <Text>  • Web Dashboard</Text>
        <Text>  • Kubernetes RBAC Configuration</Text>
        <Text>  • Ingress for HTTPS access</Text>
      </Box>
      <Box marginTop={1}>
        <Text dimColor>Press any key to continue...</Text>
      </Box>
    </Box>
  );
};
