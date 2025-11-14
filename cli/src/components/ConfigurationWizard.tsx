import React, { useState, useEffect } from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import SelectInput from 'ink-select-input';
import { generateJWTSecret, generateDatabasePassword } from '../utils/secrets.js';
import { loadConfig, saveConfig } from '../utils/config.js';

export interface Configuration {
  namespace: string;
  releaseName: string;
  ingressHost: string;
  ingressDomain: string;
  ingressClass: string;
  jwtSecret: string;
  dbPassword: string;
  installDatabase: boolean;
  dbHost?: string;
  dbPort?: string;
  dbUser?: string;
  dbName?: string;
  tlsEnabled: boolean;
  certManagerIssuer?: string;
}

interface ConfigurationWizardProps {
  onComplete: (config: Configuration) => void;
  initialConfig?: Partial<Configuration>;
  nonInteractive?: boolean;
  configFile?: string;
}

type Step =
  | 'passphrase'
  | 'namespace'
  | 'releaseName'
  | 'ingressHost'
  | 'ingressDomain'
  | 'ingressClass'
  | 'tlsEnabled'
  | 'certManagerIssuer'
  | 'installDatabase'
  | 'dbHost'
  | 'dbPort'
  | 'dbUser'
  | 'dbName'
  | 'dbPassword'
  | 'secrets'
  | 'savePassphrase'
  | 'confirm';

export const ConfigurationWizard: React.FC<ConfigurationWizardProps> = ({ onComplete, initialConfig: propInitialConfig, nonInteractive: propNonInteractive, configFile }) => {
  const [step, setStep] = useState<Step>('namespace');
  const [config, setConfig] = useState<Partial<Configuration>>({
    namespace: 'supacontrol',
    releaseName: 'supacontrol',
    ingressClass: 'nginx',
    ingressDomain: 'supabase.example.com',
    installDatabase: true, // Default to true for better out-of-box experience
    tlsEnabled: true,
    certManagerIssuer: 'letsencrypt-prod',
    dbPort: '5432',
    dbUser: 'supacontrol',
    dbName: 'supacontrol',
  });

  const [initialConfig, setInitialConfig] = useState<Partial<Configuration>>({});
  const [isNonInteractive, setIsNonInteractive] = useState(propNonInteractive || false);
  const [passphraseInput, setPassphraseInput] = useState('');
  const [savePassphraseInput, setSavePassphraseInput] = useState('');
  const [configPassphrase, setConfigPassphrase] = useState<string | undefined>(undefined);

  useEffect(() => {
    const init = async () => {
      const nonInteractive = propNonInteractive || !!process.env.NON_INTERACTIVE;
      try {
        let loadedRaw: any;
        if (nonInteractive && configFile) {
          loadedRaw = await loadConfig(configFile);
        } else {
          loadedRaw = await loadConfig();
        }
        if ('needsPassphrase' in loadedRaw && loadedRaw.needsPassphrase) {
          if (nonInteractive) {
            const envPass = process.env.SUPACTL_CONFIG_PASSPHRASE;
            if (!envPass) {
              throw new Error('Passphrase required for non-interactive mode with encrypted config. Set SUPACTL_CONFIG_PASSPHRASE.');
            }
            loadedRaw = await loadConfig(configFile || undefined, envPass);
            setConfigPassphrase(envPass);
          } else {
            setStep('passphrase');
            return;
          }
        }
        let loaded: Partial<Configuration> = {
          namespace: loadedRaw.namespace,
          releaseName: loadedRaw.releaseName,
          ingressHost: loadedRaw.ingressHost,
          ingressDomain: loadedRaw.ingressDomain,
          ingressClass: loadedRaw.ingressClass,
          tlsEnabled: loadedRaw.tlsEnabled,
          certManagerIssuer: loadedRaw.certManagerIssuer,
          installDatabase: loadedRaw.installDatabase,
          dbHost: loadedRaw.dbHost,
          dbPort: loadedRaw.dbPort,
          dbUser: loadedRaw.dbUser,
          dbName: loadedRaw.dbName,
        };
        if (loadedRaw.secrets && typeof loadedRaw.secrets === 'object' && !('encrypted' in loadedRaw.secrets)) {
          if (loadedRaw.secrets.postgresPassword) {
            loaded.dbPassword = loadedRaw.secrets.postgresPassword;
          }
          if ((loadedRaw.secrets as any).jwtSecret) {
            loaded.jwtSecret = (loadedRaw.secrets as any).jwtSecret;
          }
        }
        if (nonInteractive) {
          if (Object.keys(loaded).length === 0) {
            throw new Error('No saved configuration found. Cannot proceed in non-interactive mode.');
          }
          if (loaded.installDatabase === false) {
            if (!loaded.dbHost || !loaded.dbPort || !loaded.dbUser || !loaded.dbName || !loaded.dbPassword) {
              throw new Error('Missing required external database configuration in non-interactive mode.');
            }
          }
          setInitialConfig(loaded);
          let fullConfig = { ...config, ...loaded } as Partial<Configuration>;
          if (!fullConfig.jwtSecret) {
            fullConfig.jwtSecret = generateJWTSecret();
          }
          if (fullConfig.installDatabase && !fullConfig.dbPassword) {
            fullConfig.dbPassword = generateDatabasePassword();
          }
          setConfig(fullConfig);
          setIsNonInteractive(true);
          setStep('confirm');
          return;
        }
        setInitialConfig(loaded);
        setConfig(prev => ({ ...prev, ...loaded }));
      } catch (error: any) {
        if (nonInteractive) {
          console.error(error.message);
          process.exit(1);
        } else {
          console.warn(`Failed to load saved config: ${error.message}`);
        }
      }
    };
    init();
  }, [propNonInteractive, configFile]);

  useEffect(() => {
    if (step === 'tlsEnabled' && config.tlsEnabled !== undefined) {
      nextStep();
    }
  }, [step, config.tlsEnabled]);

  useEffect(() => {
    if (step === 'installDatabase' && config.installDatabase !== undefined) {
      if (!config.installDatabase) {
        setStep('dbHost');
      } else {
        nextStep();
      }
    }
  }, [step, config.installDatabase]);

  useEffect(() => {
    if (step === 'confirm' && isNonInteractive) {
      confirmAndContinue();
    }
  }, [step, isNonInteractive]);

  useEffect(() => {
    if (step === 'secrets' && config.jwtSecret && (config.installDatabase || config.dbPassword)) {
      generateSecrets();
    }
  }, [step, config.jwtSecret, config.dbPassword, config.installDatabase]);

  // Helper function to get missing required database fields for external database
  const getMissingDbFields = (config: Partial<Configuration>) => {
    if (config.installDatabase) {
      return [];
    }

    const requiredDbFields: Step[] = ['dbHost', 'dbUser', 'dbName', 'dbPassword'];
    return requiredDbFields.filter(field => !config[field as keyof typeof config]);
  };

  const handleInput = (field: keyof Configuration, value: any) => {
    setConfig({ ...config, [field]: value });
  };

  const nextStep = () => {
    const steps: Step[] = [
      'passphrase',
      'namespace',
      'releaseName',
      'ingressHost',
      'ingressDomain',
      'ingressClass',
      'tlsEnabled',
      ...(config.tlsEnabled === true ? ['certManagerIssuer'] : []),
      'installDatabase',
      ...(config.installDatabase === false ? ['dbHost', 'dbPort', 'dbUser', 'dbName', 'dbPassword'] : []),
      'secrets',
      'savePassphrase',
      'confirm',
    ] as Step[];

    const currentIndex = steps.indexOf(step);
    if (currentIndex < steps.length - 1) {
      setStep(steps[currentIndex + 1]);
    }
  };

  const generateSecrets = () => {
    const newConfig = { ...config };
    let needsGeneration = false;
    if (!newConfig.jwtSecret) {
      newConfig.jwtSecret = generateJWTSecret();
      needsGeneration = true;
    }
    if (config.installDatabase && !newConfig.dbPassword) {
      newConfig.dbPassword = generateDatabasePassword();
      needsGeneration = true;
    }
    if (needsGeneration) {
      console.log('Generated missing secrets.');
    } else {
      console.log('Using saved secrets.');
    }
    setConfig(newConfig);
    nextStep();
  };

  const confirmAndContinue = async () => {
    const missingFields = getMissingDbFields(config);
    if (missingFields.length > 0) {
      setStep(missingFields[0] as Step);
      return;
    }

    const currentNonSecret = { ...config };
    delete currentNonSecret.jwtSecret;
    delete currentNonSecret.dbPassword;
    const initialNonSecret = { ...initialConfig };
    delete initialNonSecret.jwtSecret;
    delete initialNonSecret.dbPassword;
    const hasChanges = JSON.stringify(currentNonSecret) !== JSON.stringify(initialNonSecret) ||
      (config.dbPassword && config.dbPassword !== initialConfig.dbPassword);

    if (hasChanges) {
      try {
        const saveObj: any = { ...currentNonSecret };
        if (config.dbPassword) {
          saveObj.secrets = {
            postgresPassword: config.dbPassword,
          };
        }
        if (config.jwtSecret) {
          if (!saveObj.secrets) saveObj.secrets = {};
          saveObj.secrets.jwtSecret = config.jwtSecret;
        }
        if (!configPassphrase) {
          console.warn('No passphrase set; saving without encryption.');
        } else {
          await saveConfig(saveObj, configPassphrase);
        }
        console.log('Updated configuration saved.');
      } catch (saveError: any) {
        console.warn(`Failed to save updated config: ${saveError.message}`);
      }
    }

    onComplete(config as Configuration);
  };

  return (
    <Box flexDirection="column" paddingY={1}>
      <Box marginBottom={1}>
        <Text bold color="cyan">
          ⚙️  Configuration Wizard
        </Text>
      </Box>

      {step === 'namespace' && (
        <Box flexDirection="column">
          <Text>Enter Kubernetes namespace (default: supacontrol):</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.namespace || ''}
              onChange={(value) => handleInput('namespace', value)}
              onSubmit={nextStep}
              placeholder="supacontrol"
            />
          </Box>
        </Box>
      )}

      {step === 'releaseName' && (
        <Box flexDirection="column">
          <Text>Enter Helm release name (default: supacontrol):</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.releaseName || ''}
              onChange={(value) => handleInput('releaseName', value)}
              onSubmit={nextStep}
              placeholder="supacontrol"
            />
          </Box>
        </Box>
      )}

      {step === 'ingressHost' && (
        <Box flexDirection="column">
          <Text>Enter the hostname for the SupaControl dashboard:</Text>
          <Text dimColor>(e.g., supacontrol.yourdomain.com)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.ingressHost || ''}
              onChange={(value) => handleInput('ingressHost', value)}
              onSubmit={nextStep}
              placeholder="supacontrol.example.com"
            />
          </Box>
        </Box>
      )}

      {step === 'ingressDomain' && (
        <Box flexDirection="column">
          <Text>Enter the base domain for Supabase instances:</Text>
          <Text dimColor>(e.g., supabase.yourdomain.com - instances will be: project.supabase.yourdomain.com)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.ingressDomain || ''}
              onChange={(value) => handleInput('ingressDomain', value)}
              onSubmit={nextStep}
              placeholder="supabase.example.com"
            />
          </Box>
        </Box>
      )}

      {step === 'ingressClass' && (
        <Box flexDirection="column">
          <Text>Enter Ingress class name (default: nginx):</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.ingressClass || ''}
              onChange={(value) => handleInput('ingressClass', value)}
              onSubmit={nextStep}
              placeholder="nginx"
            />
          </Box>
        </Box>
      )}

      {step === 'tlsEnabled' && (
        <Box flexDirection="column">
          <Text>Enable TLS/HTTPS with cert-manager?</Text>
          <Box marginTop={1}>
            <SelectInput
              items={[
                { label: 'Yes (Recommended)', value: true },
                { label: 'No', value: false },
              ]}
              onSelect={(item) => {
                handleInput('tlsEnabled', item.value);
                nextStep();
              }}
            />
          </Box>
        </Box>
      )}

      {step === 'certManagerIssuer' && (
        <Box flexDirection="column">
          <Text>Enter cert-manager ClusterIssuer name:</Text>
          <Text dimColor>(default: letsencrypt-prod)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.certManagerIssuer || ''}
              onChange={(value) => handleInput('certManagerIssuer', value)}
              onSubmit={nextStep}
              placeholder="letsencrypt-prod"
            />
          </Box>
        </Box>
      )}

      {step === 'installDatabase' && (
        <Box flexDirection="column">
          <Text>Install PostgreSQL database with SupaControl?</Text>
          <Text dimColor>(Recommended for new installations)</Text>
          <Box marginTop={1}>
            <SelectInput
              items={[
                { label: 'Yes - Install PostgreSQL (Recommended)', value: true },
                { label: 'No - Use external database', value: false },
              ]}
              onSelect={(item) => {
                const newInstallDatabase = item.value;
                handleInput('installDatabase', newInstallDatabase);

                // If switching to external database, go to dbHost step
                if (!newInstallDatabase) {
                  setStep('dbHost');
                } else {
                  nextStep();
                }
              }}
            />
          </Box>
          <Box marginTop={1}>
            <Text dimColor>• Yes: Install new PostgreSQL with SupaControl (recommended)</Text>
            <Text dimColor>• No: Use your existing PostgreSQL database (will ask for connection details)</Text>
          </Box>
        </Box>
      )}

      {step === 'dbHost' && (
        <Box flexDirection="column">
          <Text>Enter external PostgreSQL host:</Text>
          <Text dimColor>(e.g., postgres.example.com, localhost, or 192.168.1.100)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.dbHost || ''}
              onChange={(value) => handleInput('dbHost', value)}
              onSubmit={nextStep}
              placeholder="postgres.example.com"
            />
          </Box>
        </Box>
      )}

      {step === 'dbPort' && (
        <Box flexDirection="column">
          <Text>Enter PostgreSQL port:</Text>
          <Text dimColor>(default: 5432)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.dbPort || ''}
              onChange={(value) => handleInput('dbPort', value)}
              onSubmit={nextStep}
              placeholder="5432"
            />
          </Box>
        </Box>
      )}

      {step === 'dbUser' && (
        <Box flexDirection="column">
          <Text>Enter PostgreSQL username:</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.dbUser || ''}
              onChange={(value) => handleInput('dbUser', value)}
              onSubmit={nextStep}
              placeholder="supacontrol"
            />
          </Box>
        </Box>
      )}

      {step === 'dbName' && (
        <Box flexDirection="column">
          <Text>Enter PostgreSQL database name:</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.dbName || ''}
              onChange={(value) => handleInput('dbName', value)}
              onSubmit={nextStep}
              placeholder="supacontrol"
            />
          </Box>
        </Box>
      )}

      {step === 'dbPassword' && (
        <Box flexDirection="column">
          <Text>Enter the password for your existing PostgreSQL database{initialConfig.dbPassword ? ' (press Enter to use saved)' : ''}:</Text>
          <Text dimColor>(This should be the password for the database user you specified above)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value=""
              onChange={(value) => handleInput('dbPassword', value)}
              onSubmit={(inputValue) => {
                if (inputValue.trim()) {
                  handleInput('dbPassword', inputValue);
                } else if (initialConfig.dbPassword) {
                  // Use saved
                  handleInput('dbPassword', initialConfig.dbPassword);
                }
                nextStep();
              }}
              placeholder={initialConfig.dbPassword ? "Press Enter to use saved password" : "Your database password (required)"}
              showCursor={true}
            />
          </Box>
        </Box>
      )}

      {step === 'secrets' && (
        <Box flexDirection="column">
          <Text bold color="yellow">
            🔐 Generating secure secrets...
          </Text>
          <Box marginTop={1}>
            {config.installDatabase ? (
              <Text>Press Enter to generate JWT secret and database password</Text>
            ) : (
              <Text>Press Enter to generate JWT secret (database password already provided)</Text>
            )}
          </Box>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput value="" onChange={(value: string) => {}} onSubmit={generateSecrets} showCursor={false} />
          </Box>
        </Box>
      )}

      {step === 'confirm' && (
        <Box flexDirection="column">
          <Text bold color="cyan">
            📝 Configuration Summary
          </Text>
          <Box marginTop={1} flexDirection="column" borderStyle="round" borderColor="gray" paddingX={2} paddingY={1}>
            <Text>
              <Text color="gray">Namespace: </Text>
              <Text bold>{config.namespace}</Text>
            </Text>
            <Text>
              <Text color="gray">Release Name: </Text>
              <Text bold>{config.releaseName}</Text>
            </Text>
            <Text>
              <Text color="gray">Dashboard URL: </Text>
              <Text bold color="cyan">
                {config.tlsEnabled ? 'https' : 'http'}://{config.ingressHost}
              </Text>
            </Text>
            <Text>
              <Text color="gray">Supabase Domain: </Text>
              <Text bold>{config.ingressDomain}</Text>
            </Text>
            <Text>
              <Text color="gray">Ingress Class: </Text>
              <Text bold>{config.ingressClass}</Text>
            </Text>
            <Text>
              <Text color="gray">TLS Enabled: </Text>
              <Text bold color={config.tlsEnabled ? 'green' : 'yellow'}>{config.tlsEnabled ? 'Yes' : 'No'}</Text>
            </Text>
            {config.tlsEnabled && (
              <Text>
                <Text color="gray">Cert Manager Issuer: </Text>
                <Text bold>{config.certManagerIssuer}</Text>
              </Text>
            )}
            <Text>
              <Text color="gray">Database: </Text>
              <Text bold>
                {config.installDatabase ? 'Install PostgreSQL' :
                  `External (${config.dbHost || 'undefined'}:${config.dbPort || '5432'}/${config.dbName || 'supacontrol'})`}
              </Text>
            </Text>
            {!config.installDatabase && (
              <>
                <Text>
                  <Text color="gray">DB User: </Text>
                  <Text bold>{config.dbUser || 'supacontrol'}</Text>
                </Text>
                <Text>
                  <Text color="gray">DB Password: </Text>
                  {!config.dbPassword ? (
                    <Text bold color="red">MISSING - Please provide database password</Text>
                  ) : (
                    <Text bold color={initialConfig.dbPassword && config.dbPassword === initialConfig.dbPassword ? "green" : "yellow"}>
                      {initialConfig.dbPassword && config.dbPassword === initialConfig.dbPassword ? "Loaded from config ✓" : "Provided by user ✓"}
                    </Text>
                  )}
                </Text>
              </>
            )}
            {config.installDatabase && (
              <Text>
                <Text color="gray">DB Password: </Text>
                <Text bold color="green">Generated ✓</Text>
              </Text>
            )}
            <Text>
              <Text color="gray">JWT Secret: </Text>
              <Text bold color={initialConfig.jwtSecret ? "green" : "yellow"}>{initialConfig.jwtSecret ? "Loaded from config ✓" : "Generated ✓"}</Text>
            </Text>
          </Box>

          {/* Validation warnings */}
          {!config.installDatabase && (() => {
            const missingFields = getMissingDbFields(config);

            if (missingFields.length === 0) return null;

            const fieldDescriptions: Record<string, string> = {
              'dbHost': 'Database host address',
              'dbUser': 'Database username',
              'dbName': 'Database name',
              'dbPassword': 'Database password'
            };

            return (
              <Box marginTop={1} flexDirection="column" borderStyle="round" borderColor="red" paddingX={2} paddingY={1}>
                <Text bold color="red">
                  ⚠️  Incomplete External Database Configuration
                </Text>
                <Box marginTop={1}>
                  <Text color="red">
                    The following fields are required for external database:
                  </Text>
                </Box>
                {missingFields.map(field => (
                  <Text key={field} color="red">• {fieldDescriptions[field]}</Text>
                ))}
                <Box marginTop={1}>
                  <Text color="red">Press Enter to go to database configuration...</Text>
                </Box>
                <Box marginTop={1}>
                  <Text color="yellow">Note: You can go back to fix any step</Text>
                </Box>
              </Box>
            );
          })()}

          <Box marginTop={1}>
            <Text>
              {!config.installDatabase && (() => {
                const missingFields = getMissingDbFields(config);
                return missingFields.length > 0
                  ? 'Press Enter to fix database configuration...'
                  : 'Press Enter to continue with installation...';
              })()}
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput value="" onChange={(value: string) => {}} onSubmit={confirmAndContinue} showCursor={false} />
          </Box>
        </Box>
      )}

      {step === 'passphrase' && (
        <Box flexDirection="column">
          <Text bold color="yellow">🔐 Enter passphrase for existing configuration</Text>
          <Text dimColor>(To decrypt saved secrets)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={passphraseInput}
              onChange={setPassphraseInput}
              onSubmit={async (value) => {
                if (!value.trim()) {
                  return;
                }
                try {
                  const loadedWithPass = await loadConfig(undefined, value);
                  let loaded: Partial<Configuration> = {
                    namespace: loadedWithPass.namespace,
                    releaseName: loadedWithPass.releaseName,
                    ingressHost: loadedWithPass.ingressHost,
                    ingressDomain: loadedWithPass.ingressDomain,
                    ingressClass: loadedWithPass.ingressClass,
                    tlsEnabled: loadedWithPass.tlsEnabled,
                    certManagerIssuer: loadedWithPass.certManagerIssuer,
                    installDatabase: loadedWithPass.installDatabase,
                    dbHost: loadedWithPass.dbHost,
                    dbPort: loadedWithPass.dbPort,
                    dbUser: loadedWithPass.dbUser,
                    dbName: loadedWithPass.dbName,
                  };
                  if (loadedWithPass.secrets && typeof loadedWithPass.secrets === 'object' && !('encrypted' in loadedWithPass.secrets)) {
                    if (loadedWithPass.secrets.postgresPassword) {
                      loaded.dbPassword = loadedWithPass.secrets.postgresPassword;
                    }
                    if ((loadedWithPass.secrets as any).jwtSecret) {
                      loaded.jwtSecret = (loadedWithPass.secrets as any).jwtSecret;
                    }
                  }
                  setInitialConfig(loaded);
                  setConfig(prev => ({ ...prev, ...loaded }));
                  setConfigPassphrase(value);
                  setPassphraseInput('');
                  setStep('namespace');
                } catch (error: any) {
                  console.log('\n❌ Invalid passphrase or config error: ' + error.message + '\n');
                  setPassphraseInput('');
                }
              }}
              placeholder="Enter passphrase"
              showCursor={true}
            />
          </Box>
          <Box marginTop={1}>
            <Text dimColor>If this is a new installation, delete ~/.supacontrol/config.json and restart.</Text>
          </Box>
        </Box>
      )}

      {step === 'savePassphrase' && (
        <Box flexDirection="column">
          <Text bold color="yellow">🔐 Set passphrase for config encryption</Text>
          {configPassphrase ? (
            <>
              <Text>Using existing passphrase from loaded config.</Text>
              <Box marginTop={1}>
                <Text color="green">➜ Press Enter to continue</Text>
                <TextInput value="" onChange={() => {}} onSubmit={nextStep} showCursor={false} />
              </Box>
            </>
          ) : (
            <>
              <Text>Choose a passphrase to encrypt your config secrets.</Text>
              <Text dimColor>(Keep it safe, you'll need it for future updates)</Text>
              <Box marginTop={1}>
                <Text color="green">➜ </Text>
                <TextInput
                  value={savePassphraseInput}
                  onChange={setSavePassphraseInput}
                  onSubmit={(value) => {
                    if (!value.trim()) return;
                    setConfigPassphrase(value);
                    setSavePassphraseInput('');
                    nextStep();
                  }}
                  placeholder="Choose a secure passphrase"
                  showCursor={true}
                />
              </Box>
            </>
          )}
        </Box>
      )}
    </Box>
  );
};
