import React, { useState } from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import SelectInput from 'ink-select-input';
import { generateJWTSecret, generateDatabasePassword } from '../utils/secrets.js';

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
}

type Step =
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
  | 'confirm';

export const ConfigurationWizard: React.FC<ConfigurationWizardProps> = ({ onComplete }) => {
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

  const handleInput = (field: keyof Configuration, value: any) => {
    setConfig({ ...config, [field]: value });
  };

  const nextStep = () => {
    const steps: Step[] = [
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
      'confirm',
    ] as Step[];

    const currentIndex = steps.indexOf(step);
    if (currentIndex < steps.length - 1) {
      setStep(steps[currentIndex + 1]);
    }
  };

  const generateSecrets = () => {
    const newConfig = {
      ...config,
      jwtSecret: generateJWTSecret(),
    };
    
    // Only generate new database password if installing a new database
    if (config.installDatabase) {
      newConfig.dbPassword = generateDatabasePassword();
    }
    
    setConfig(newConfig);
    nextStep();
  };

  const confirmAndContinue = () => {
    // Validate required fields for external database
    if (!config.installDatabase) {
      const requiredDbFields: Step[] = ['dbHost', 'dbUser', 'dbName', 'dbPassword'];
      for (const field of requiredDbFields) {
        if (!config[field as keyof typeof config]) {
          setStep(field);
          return;
        }
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
          <Text>Enter the password for your existing PostgreSQL database:</Text>
          <Text dimColor>(This should be the password for the database user you specified above)</Text>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput
              value={config.dbPassword || ''}
              onChange={(value) => handleInput('dbPassword', value)}
              onSubmit={nextStep}
              placeholder="Your database password"
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
                    <Text bold color="yellow">Provided by user ✓</Text>
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
              <Text bold color="green">Generated ✓</Text>
            </Text>
          </Box>
          
          {/* Validation warnings */}
          {!config.installDatabase && (!config.dbHost || !config.dbUser || !config.dbName || !config.dbPassword) && (
            <Box marginTop={1} flexDirection="column" borderStyle="round" borderColor="red" paddingX={2} paddingY={1}>
              <Text bold color="red">
                ⚠️  Incomplete External Database Configuration
              </Text>
              <Box marginTop={1}>
                <Text color="red">
                  The following fields are required for external database:
                </Text>
              </Box>
              {!config.dbHost && <Text color="red">• Database host address</Text>}
              {!config.dbUser && <Text color="red">• Database username</Text>}
              {!config.dbName && <Text color="red">• Database name</Text>}
              {!config.dbPassword && <Text color="red">• Database password</Text>}
              <Box marginTop={1}>
                <Text color="red">Press Enter to go to database configuration...</Text>
              </Box>
              <Box marginTop={1}>
                <Text color="yellow">Note: You can go back to fix any step</Text>
              </Box>
            </Box>
          )}
          
          <Box marginTop={1}>
            <Text>
              {!config.installDatabase && (!config.dbHost || !config.dbUser || !config.dbName || !config.dbPassword)
                ? 'Press Enter to fix database configuration...'
                : 'Press Enter to continue with installation...'
              }
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text color="green">➜ </Text>
            <TextInput value="" onChange={(value: string) => {}} onSubmit={confirmAndContinue} showCursor={false} />
          </Box>
        </Box>
      )}
    </Box>
  );
};
