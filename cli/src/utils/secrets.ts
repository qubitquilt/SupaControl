import { randomBytes } from 'crypto';
import { customAlphabet } from 'nanoid';

/**
 * Generate a secure random string
 */
export function generateSecureSecret(length: number = 64): string {
  return randomBytes(length).toString('base64').slice(0, length);
}

/**
 * Generate a JWT secret (base64 encoded)
 */
export function generateJWTSecret(): string {
  return randomBytes(64).toString('base64');
}

/**
 * Generate a database password
 */
export function generateDatabasePassword(length: number = 32): string {
  const alphabet = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
  const nanoid = customAlphabet(alphabet, length);
  return nanoid();
}

/**
 * Generate an API key
 */
export function generateAPIKey(): string {
  const key = randomBytes(32).toString('base64url');
  return `sk_${key}`;
}

/**
 * Validate secret strength
 */
export function validateSecretStrength(secret: string, minLength: number = 32): { valid: boolean; message?: string } {
  if (secret.length < minLength) {
    return { valid: false, message: `Secret must be at least ${minLength} characters long` };
  }

  // Check for character diversity
  const hasLower = /[a-z]/.test(secret);
  const hasUpper = /[A-Z]/.test(secret);
  const hasNumber = /[0-9]/.test(secret);
  const hasSpecial = /[^a-zA-Z0-9]/.test(secret);

  const diversity = [hasLower, hasUpper, hasNumber, hasSpecial].filter(Boolean).length;

  if (diversity < 3) {
    return { valid: false, message: 'Secret should contain a mix of uppercase, lowercase, numbers, and special characters' };
  }

  return { valid: true };
}
