import { describe, it, expect } from 'vitest';
import {
  generateSecureSecret,
  generateJWTSecret,
  generateDatabasePassword,
  generateAPIKey,
  validateSecretStrength,
} from './secrets.js';

describe('secrets utilities', () => {
  describe('generateSecureSecret', () => {
    it('should generate a secret of default length 64', () => {
      const secret = generateSecureSecret();
      expect(secret).toHaveLength(64);
    });

    it('should generate a secret of specified length', () => {
      const length = 32;
      const secret = generateSecureSecret(length);
      expect(secret).toHaveLength(length);
    });

    it('should generate different secrets on each call', () => {
      const secret1 = generateSecureSecret();
      const secret2 = generateSecureSecret();
      expect(secret1).not.toBe(secret2);
    });

    it('should generate base64-compatible characters', () => {
      const secret = generateSecureSecret(64);
      expect(secret).toMatch(/^[A-Za-z0-9+/]+$/);
    });
  });

  describe('generateJWTSecret', () => {
    it('should generate a base64-encoded secret', () => {
      const secret = generateJWTSecret();
      expect(secret).toBeTruthy();
      // Base64 strings are typically longer than the raw bytes
      expect(secret.length).toBeGreaterThan(64);
    });

    it('should generate different secrets on each call', () => {
      const secret1 = generateJWTSecret();
      const secret2 = generateJWTSecret();
      expect(secret1).not.toBe(secret2);
    });

    it('should be a valid base64 string', () => {
      const secret = generateJWTSecret();
      expect(() => Buffer.from(secret, 'base64')).not.toThrow();
    });
  });

  describe('generateDatabasePassword', () => {
    it('should generate a password of default length 32', () => {
      const password = generateDatabasePassword();
      expect(password).toHaveLength(32);
    });

    it('should generate a password of specified length', () => {
      const length = 16;
      const password = generateDatabasePassword(length);
      expect(password).toHaveLength(length);
    });

    it('should generate different passwords on each call', () => {
      const password1 = generateDatabasePassword();
      const password2 = generateDatabasePassword();
      expect(password1).not.toBe(password2);
    });

    it('should contain only allowed characters', () => {
      const password = generateDatabasePassword(100);
      expect(password).toMatch(/^[a-zA-Z0-9!@#$%^&*]+$/);
    });

    it('should contain diverse character types', () => {
      // Generate a long password to ensure diversity
      const password = generateDatabasePassword(100);
      expect(password).toMatch(/[a-z]/); // lowercase
      expect(password).toMatch(/[A-Z]/); // uppercase
      expect(password).toMatch(/[0-9]/); // numbers
    });
  });

  describe('generateAPIKey', () => {
    it('should generate a key with sk_ prefix', () => {
      const apiKey = generateAPIKey();
      expect(apiKey).toMatch(/^sk_/);
    });

    it('should generate different keys on each call', () => {
      const key1 = generateAPIKey();
      const key2 = generateAPIKey();
      expect(key1).not.toBe(key2);
    });

    it('should use base64url encoding', () => {
      const apiKey = generateAPIKey();
      const keyPart = apiKey.replace('sk_', '');
      // base64url should not contain +, /, or =
      expect(keyPart).not.toMatch(/[+/=]/);
    });
  });

  describe('validateSecretStrength', () => {
    it('should validate a strong secret', () => {
      const strongSecret = 'MyStr0ngP@ssw0rd!WithL0tsOfCh@rs';
      const result = validateSecretStrength(strongSecret);
      expect(result.valid).toBe(true);
      expect(result.message).toBeUndefined();
    });

    it('should reject a secret that is too short', () => {
      const shortSecret = 'Short123!';
      const result = validateSecretStrength(shortSecret, 32);
      expect(result.valid).toBe(false);
      expect(result.message).toContain('at least 32 characters');
    });

    it('should reject a secret with insufficient diversity', () => {
      const weakSecret = 'a'.repeat(50); // Only lowercase
      const result = validateSecretStrength(weakSecret);
      expect(result.valid).toBe(false);
      expect(result.message).toContain('mix of uppercase, lowercase, numbers');
    });

    it('should accept a secret with only 3 character types', () => {
      const goodSecret = 'MyStrongPassword123WithNoSpecial'.padEnd(32, '0');
      const result = validateSecretStrength(goodSecret);
      expect(result.valid).toBe(true);
    });

    it('should work with custom minimum length', () => {
      const secret = 'MyStr0ng!Pass';
      const result = validateSecretStrength(secret, 10);
      expect(result.valid).toBe(true);
    });

    it('should validate generated database password', () => {
      const password = generateDatabasePassword(32);
      const result = validateSecretStrength(password);
      expect(result.valid).toBe(true);
    });
  });
});
