#!/usr/bin/env node
import React, { useState } from 'react';
import { render, Box, Text, useInput } from 'ink';
import { Welcome } from './components/Welcome.js';
import { PrerequisitesCheck } from './components/PrerequisitesCheck.js';
import { ConfigurationWizard, Configuration } from './components/ConfigurationWizard.js';
import { Installation } from './components/Installation.js';
import { Complete } from './components/Complete.js';

// Timing constants for transitions and delays
const SUCCESS_TRANSITION_DELAY_MS = 1500;
const ERROR_EXIT_DELAY_MS = 3000;

type Screen = 'welcome' | 'prerequisites' | 'configuration' | 'installation' | 'complete';

const App: React.FC = () => {
  const [screen, setScreen] = useState<Screen>('welcome');
  const [prerequisitesPassed, setPrerequisitesPassed] = useState(false);
  const [configuration, setConfiguration] = useState<Configuration | null>(null);
  const [installationSuccess, setInstallationSuccess] = useState(false);

  // Handle welcome screen - any key press continues
  useInput((input, key) => {
    if (screen === 'welcome' && !key.ctrl) {
      setScreen('prerequisites');
    }
  });

  const handlePrerequisitesComplete = (success: boolean) => {
    setPrerequisitesPassed(success);
    if (success) {
      setTimeout(() => setScreen('configuration'), SUCCESS_TRANSITION_DELAY_MS);
    } else {
      setTimeout(() => process.exit(1), ERROR_EXIT_DELAY_MS);
    }
  };

  const handleConfigurationComplete = (config: Configuration) => {
    setConfiguration(config);
    setScreen('installation');
  };

  const handleInstallationComplete = (success: boolean) => {
    setInstallationSuccess(success);
    setScreen('complete');
  };

  return (
    <Box flexDirection="column" padding={1}>
      {screen === 'welcome' && <Welcome />}

      {screen === 'prerequisites' && (
        <PrerequisitesCheck onComplete={handlePrerequisitesComplete} />
      )}

      {screen === 'configuration' && (
        <ConfigurationWizard onComplete={handleConfigurationComplete} />
      )}

      {screen === 'installation' && configuration && (
        <Installation config={configuration} onComplete={handleInstallationComplete} />
      )}

      {screen === 'complete' && configuration && (
        <Complete config={configuration} success={installationSuccess} />
      )}

      {/* Footer */}
      <Box marginTop={1} borderStyle="single" borderColor="gray" paddingX={1}>
        <Text dimColor>
          SupaControl Installer v0.1.0 | Press Ctrl+C to exit
        </Text>
      </Box>
    </Box>
  );
};

// Render the app
const { waitUntilExit } = render(<App />);

// Handle exit
waitUntilExit().then(() => {
  process.exit(0);
});
