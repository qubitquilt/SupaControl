#!/usr/bin/env node
import React, { useState, useEffect } from 'react';
import { generateJWTSecret, generateDatabasePassword } from './utils/secrets.js';
import { render, Box, Text, useInput } from 'ink';
import { loadConfig, saveConfig, ConfigType } from './utils/config.js';
import { Welcome } from './components/Welcome.js';
import { PrerequisitesCheck } from './components/PrerequisitesCheck.js';
import { ConfigurationWizard, Configuration } from './components/ConfigurationWizard.js';
import { Installation } from './components/Installation.js';
import { Complete } from './components/Complete.js';
import { checkRelease, checkSupabaseInstance, generateHelmValues, saveHelmValues, installHelm, applySupabaseInstance, applySecrets, generateSupabaseInstance } from './utils/helm.js';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { execa } from 'execa';
import { access, constants } from 'fs/promises';

// Timing constants for transitions and delays
const SUCCESS_TRANSITION_DELAY_MS = 1500;
const ERROR_EXIT_DELAY_MS = 3000;

type Screen = 'welcome' | 'prerequisites' | 'configuration' | 'installation' | 'complete';

interface AppProps {
  nonInteractive?: boolean;
  configFile?: string;
  dryRun?: boolean;
  initialConfig?: Partial<Configuration>;
}

const App: React.FC<AppProps> = ({ nonInteractive = false, configFile, dryRun = false, initialConfig }) => {
  const [screen, setScreen] = useState<Screen>('welcome');
  const [prerequisitesPassed, setPrerequisitesPassed] = useState(false);
  const [configuration, setConfiguration] = useState<Configuration | null>(null);
  const [installationSuccess, setInstallationSuccess] = useState(false);

  // Parse command line arguments
  const args = process.argv.slice(2);
  let parsedNonInteractive = nonInteractive;
  let parsedConfigFile = configFile;
  let parsedDryRun = dryRun;
  let loadedConfig: Partial<Configuration> | null = initialConfig || null;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--non-interactive' || arg === '-n') {
      parsedNonInteractive = true;
    } else if (arg === '--config-file' || arg === '-c') {
      if (i + 1 < args.length) {
        parsedConfigFile = args[++i];
      } else {
        console.error('Error: --config-file requires a file path.');
        process.exit(1);
      }
    } else if (arg === '--dry-run' || arg === '-d') {
      parsedDryRun = true;
    }
  }

  // Load config if non-interactive or config-file specified
  useEffect(() => {
    const loadInitialConfig = async () => {
      if (parsedNonInteractive || parsedConfigFile) {
        try {
          const configPath = parsedConfigFile || undefined;
          const config = await loadConfig(configPath);
          if (Object.keys(config).length === 0 && parsedNonInteractive && !parsedConfigFile) {
            console.error('Error: No saved config found; run interactively first.');
            process.exit(1);
          }
          // Merge with defaults if needed, but loadConfig returns partial
          loadedConfig = config as Partial<Configuration>;
          // Generate missing secrets if needed
          if (!loadedConfig.jwtSecret) {
            loadedConfig.jwtSecret = require('./utils/secrets.js').generateJWTSecret();
          }
          if (loadedConfig.installDatabase && !loadedConfig.dbPassword) {
            loadedConfig.dbPassword = require('./utils/secrets.js').generateDatabasePassword();
          }
          // For external DB, ensure required fields
          if (loadedConfig.installDatabase === false) {
            const requiredFields = ['dbHost', 'dbPort', 'dbUser', 'dbName', 'dbPassword'];
            const missing = requiredFields.filter(field => !loadedConfig![field as keyof Configuration]);
            if (missing.length > 0) {
              console.error(`Error: Missing required fields for external database: ${missing.join(', ')}`);
              process.exit(1);
            }
          }
        } catch (error: any) {
          console.error(`Error loading config: ${error.message}`);
          process.exit(1);
        }
      }
    };

    loadInitialConfig();
  }, [parsedNonInteractive, parsedConfigFile]);

  // Handle welcome screen - any key press continues
  useInput((input, key) => {
    if (screen === 'welcome' && !key.ctrl) {
      setScreen('prerequisites');
    }
  });

  // If non-interactive, skip welcome after init
  useEffect(() => {
    if (parsedNonInteractive && screen === 'welcome') {
      setTimeout(() => setScreen('prerequisites'), 500);
    }
  }, [screen, parsedNonInteractive]);

  const handlePrerequisitesComplete = (success: boolean) => {
    setPrerequisitesPassed(success);
    if (success) {
      if (parsedNonInteractive && loadedConfig) {
        // Skip configuration, go directly to installation
        setConfiguration(loadedConfig as Configuration);
        setTimeout(() => setScreen('installation'), SUCCESS_TRANSITION_DELAY_MS);
      } else {
        setTimeout(() => setScreen('configuration'), SUCCESS_TRANSITION_DELAY_MS);
      }
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

  // Pass initial config and flags to ConfigurationWizard if interactive
  const wizardProps = parsedNonInteractive && loadedConfig
    ? { initialConfig: loadedConfig, nonInteractive: true, configFile: parsedConfigFile }
    : { configFile: parsedConfigFile };

  return (
    <Box flexDirection="column" padding={1}>
      {screen === 'welcome' && <Welcome />}

      {screen === 'prerequisites' && (
        <PrerequisitesCheck onComplete={handlePrerequisitesComplete} />
      )}

      {screen === 'configuration' && (
        <ConfigurationWizard {...wizardProps} onComplete={handleConfigurationComplete} />
      )}

      {screen === 'installation' && configuration && (
        <Installation config={configuration} dryRun={parsedDryRun} onComplete={handleInstallationComplete} />
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

// Reboot command implementation
async function runRebootCommand(args: string[], configFile?: string, dryRun = false) {
  const namespace = 'supacontrol-system'; // Hardcoded as per context, or from config
  const releaseName = 'supacontrol';
  const instanceName = config.instanceName || 'supacontrol-instance';

  try {
    console.log('🔄 Starting SupaControl reboot...');
    console.log(`Namespace: ${namespace}`);
    console.log(`Release: ${releaseName}`);
    console.log(`Instance: ${instanceName}`);

    // Load config
    let config: Partial<ConfigType>;
    if (configFile) {
      config = await loadConfig(configFile);
    } else {
      config = await loadConfig();
    }

    if (Object.keys(config).length === 0) {
      console.error('Error: No saved configuration found. Please run the interactive wizard first to create a config.');
      process.exit(1);
    }

    console.log('✓ Configuration loaded successfully');

    // Ensure required fields and generate missing secrets
    // Assume secrets are decrypted; if not, passphrase is handled in loadConfig
    const secrets = config.secrets || {};
    if (!secrets.jwtSecret) {
      secrets.jwtSecret = generateJWTSecret();
      console.log('Generated new JWT secret');
    }
    if (!secrets.postgresPassword) {
      secrets.postgresPassword = generateDatabasePassword();
      console.log('Generated new database password');
    }
    config.secrets = secrets;

    // For helm, map to helmConfig (assume some defaults or from config)
    const helmConfig: any = {
      jwtSecret: secrets.jwtSecret,
      dbPassword: secrets.postgresPassword,
      // Assume internal DB for simplicity, or add fields if in config
      installDatabase: true, // Default
      ingressEnabled: true, // Default
      ingressHost: 'supacontrol.example.com', // Default or from config if extended
      ingressClass: 'nginx',
      ingressDomain: 'supabase.example.com',
      tlsEnabled: true,
      certManagerIssuer: 'letsencrypt-prod',
      imageTag: config.version || 'latest',
    };

    // Step 1: Check and apply secrets
    console.log('\n🔑 Applying secrets...');
    const secretsResult = await applySecrets(helmConfig, namespace, dryRun);
    if (!secretsResult.success) {
      throw new Error(secretsResult.error || 'Failed to apply secrets');
    }

    // Step 2: Check Helm release state
    console.log('\n📦 Checking Helm release state...');
    const releaseState = await checkRelease(namespace);
    console.log(`Current release exists: ${releaseState.exists}, version: ${releaseState.version || 'none'}`);

    // Always reapply Helm to ensure config is current
    const helmNeedsUpdate = true;
    console.log(`Reapplying Helm release (exists: ${releaseState.exists})`);

    // Step 3: Generate and apply Helm if needed
    if (helmNeedsUpdate) {
      console.log('\n📦 Applying Helm release...');
      const outputDir = join(dirname(fileURLToPath(import.meta.url)), '..', 'tmp'); // Temp dir relative to src
      const valuesPath = await saveHelmValues(helmConfig, outputDir);

      // Find chart path (reuse logic from Installation)
      const chartPath = await findChartPath(); // Implement below

      const helmResult = await installHelm(namespace, releaseName, valuesPath, chartPath, dryRun);
      if (!helmResult.success) {
        throw new Error(helmResult.error || 'Helm apply failed');
      }
      console.log('✓ Helm release applied successfully');
    } else {
      console.log('✓ Helm release up to date, skipping');
    }

    // Step 4: Check SupabaseInstance CR
    console.log('\n🔮 Checking SupabaseInstance CR...');
    const crState = await checkSupabaseInstance(namespace, instanceName);
    console.log(`SupabaseInstance exists: ${crState.exists}`);

    let crNeedsUpdate = !crState.exists;
    // For simplicity, always reapply if exists but to check changes, compare spec (omitted for now)
    if (crState.exists) {
      console.log('SupabaseInstance exists, reapplying to ensure config match...');
      crNeedsUpdate = true;
    }

    if (crNeedsUpdate) {
      console.log('\n🔮 Applying SupabaseInstance CR...');
      const yamlContent = generateSupabaseInstance(config);
      const crResult = await applySupabaseInstance(yamlContent, namespace, dryRun);
      if (!crResult.success) {
        throw new Error(crResult.error || 'CR apply failed');
      }
      console.log('✓ SupabaseInstance CR applied successfully');
    } else {
      console.log('✓ SupabaseInstance CR up to date, skipping');
    }

    console.log('\n✅ Reboot completed successfully!');
    console.log('Your SupaControl configuration has been reloaded and reapplied to the cluster.');
    process.exit(0);

  } catch (error: any) {
    console.error('\n❌ Reboot failed:', error.message);
    process.exit(1);
  }
}

// Helper to find chart path (extracted from Installation logic)
async function findChartPath(): Promise<string> {
  const envChartPath = process.env.SUPACONTROL_CHART_PATH;
  if (envChartPath) {
    try {
      await access(envChartPath, constants.R_OK);
      return envChartPath;
    } catch {}
  }

  const localChartPath = join(process.cwd(), 'charts/supacontrol');
  try {
    await access(localChartPath, constants.R_OK);
    return localChartPath;
  } catch {}

  const parentChartPath = join(process.cwd(), '../charts/supacontrol');
  try {
    await access(parentChartPath, constants.R_OK);
    return parentChartPath;
  } catch {}

  // Clone if needed (simplified, assume exists or error)
  throw new Error('Helm chart not found. Set SUPACONTROL_CHART_PATH or run from repo root.');
}

// Parse args and check for reboot
const args = process.argv.slice(2);
const command = args[0];
if (command === 'reboot') {
  // Parse reboot options
  let rebootConfigFile: string | undefined;
  let rebootDryRun = false;
  for (let i = 1; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--config-file' || arg === '-c') {
      if (i + 1 < args.length) {
        rebootConfigFile = args[++i];
      } else {
        console.error('Error: --config-file requires a file path.');
        process.exit(1);
      }
    } else if (arg === '--dry-run' || arg === '-d') {
      rebootDryRun = true;
    }
  }
  runRebootCommand(args, rebootConfigFile, rebootDryRun);
}

// Render the app
const { waitUntilExit } = render(<App nonInteractive={false} configFile={undefined} dryRun={false} />);

// Handle exit
waitUntilExit().then(() => {
  process.exit(0);
});
