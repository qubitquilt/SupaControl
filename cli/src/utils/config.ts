import * as fs from 'fs/promises';
import * as path from 'path';
import * as os from 'os';
import * as crypto from 'crypto';
import * as readline from 'readline';

export interface ConfigType {
  namespace?: string;
  releaseName?: string;
  ingressHost?: string;
  ingressDomain?: string;
  ingressClass?: string;
  installDatabase?: boolean;
  dbHost?: string;
  dbPort?: string;
  dbUser?: string;
  dbName?: string;
  tlsEnabled?: boolean;
  certManagerIssuer?: string;
  instanceName?: string;
  version?: string;
  size?: 'small' | 'medium' | 'large';
  components?: {
    auth?: boolean;
    // Additional components can be added here as needed
  };
  secrets?: {
    postgresPassword?: string;
    jwtSecret?: string;
    // Additional secrets can be added here as needed
  };
}

const CONFIG_DIR = path.join(os.homedir(), '.supacontrol');
const CONFIG_PATH = path.join(CONFIG_DIR, 'config.json');

async function getPassphrase(interactive: boolean = true): Promise<string> {
  let passphrase = process.env.SUPACTL_CONFIG_PASSPHRASE;
  if (!passphrase && interactive) {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    passphrase = await new Promise<string>((resolve) => {
      rl.question('Enter passphrase for config encryption: ', (pass) => {
        rl.close();
        resolve(pass);
      });
    });
    if (!passphrase.trim()) {
      throw new Error('Passphrase is required for encryption.');
    }
  }
  if (!passphrase) {
    throw new Error('Passphrase is required for encryption. Set SUPACTL_CONFIG_PASSPHRASE or run interactively.');
  }
  return passphrase;
}

async function deriveKey(passphrase: string, salt: Buffer): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    crypto.scrypt(passphrase, salt, 32, { cost: 16384 }, (err, derivedKey) => {
      if (err) reject(err);
      else resolve(derivedKey);
    });
  });
}

async function encrypt(text: string, passphrase: string): Promise<string> {
  const salt = crypto.randomBytes(16);
  const iv = crypto.randomBytes(16);
  const key = await deriveKey(passphrase, salt);
  const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
  let encrypted = cipher.update(text, 'utf8', 'hex');
  encrypted += cipher.final('hex');
  const authTag = cipher.getAuthTag();
  const combined = Buffer.concat([salt, iv, Buffer.from(encrypted, 'hex'), authTag]);
  return combined.toString('hex');
}

async function decrypt(encryptedHex: string, passphrase: string): Promise<string> {
  const combined = Buffer.from(encryptedHex, 'hex');
  if (combined.length < 48) {
    throw new Error('Invalid encrypted data.');
  }
  const salt = combined.slice(0, 16);
  const iv = combined.slice(16, 32);
  const encrypted = combined.slice(32, -16);
  const authTag = combined.slice(-16);
  const key = await deriveKey(passphrase, salt);
  const decipher = crypto.createDecipheriv('aes-256-gcm', key, iv);
  decipher.setAuthTag(authTag);
  let decrypted = decipher.update(encrypted);
  const final = decipher.final();
  if (final) {
    decrypted = Buffer.concat([decrypted, final]);
  }
  return decrypted.toString();
}

export async function decryptSecrets(encryptedHex: string, passphrase: string): Promise<any> {
  try {
    const decryptedJson = await decrypt(encryptedHex, passphrase);
    return JSON.parse(decryptedJson);
  } catch (error: any) {
    throw new Error(`Decryption failed: ${error.message}`);
  }
}

export async function loadConfig(path?: string, passphrase?: string, interactive: boolean = true): Promise<ConfigType & { needsPassphrase?: boolean }> {
  const configPath = path ? path : CONFIG_PATH;
  try {
    if (!path) {
      await fs.mkdir(CONFIG_DIR, { recursive: true });
    }
    try {
      const data = await fs.readFile(configPath, 'utf8');
      let config: ConfigType = JSON.parse(data);
      if (config.secrets && typeof config.secrets === 'object' && 'encrypted' in config.secrets) {
        if (passphrase) {
          try {
            const encrypted = (config.secrets as any).encrypted;
            const decryptedJson = await decrypt(encrypted, passphrase);
            config.secrets = JSON.parse(decryptedJson) as ConfigType['secrets'];
          } catch (decErr: any) {
            console.warn(`Failed to decrypt secrets: ${decErr.message}. Proceeding without saved secrets.`);
            config.secrets = undefined;
          }
        } else if (interactive) {
          (config as any).needsPassphrase = true;
        } else {
          throw new Error('Passphrase is required for decryption.');
        }
      }
      return config;
    } catch (readErr: any) {
      if (readErr.code === 'ENOENT') {
        return {} as ConfigType;
      }
      throw new Error(`Failed to read config file at ${configPath}: ${readErr.message}`);
    }
  } catch (error: any) {
    console.error('Error loading config:', error.message);
    return {} as ConfigType;
  }
}

export async function saveConfig(config: ConfigType, passphrase: string): Promise<void> {
  try {
    await fs.mkdir(CONFIG_DIR, { recursive: true });
    const saveConfig = { ...config };
    if (saveConfig.secrets) {
      const secretsJson = JSON.stringify(saveConfig.secrets);
      const encryptedSecrets = await encrypt(secretsJson, passphrase);
      saveConfig.secrets = { encrypted: encryptedSecrets } as any;
    }
    const json = JSON.stringify(saveConfig, null, 2);
    await fs.writeFile(CONFIG_PATH, json, 'utf8');
  } catch (error: any) {
    console.error('Error saving config:', error.message);
    throw error;
  }
}
