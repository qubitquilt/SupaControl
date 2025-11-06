import { describe, it, expect, beforeEach, vi } from 'vitest';
import axios from 'axios';
import { authAPI, instancesAPI } from './api';

// Mock axios
vi.mock('axios');

describe('API Configuration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  describe('authAPI', () => {
    it('should call login endpoint', async () => {
      const mockResponse = { data: { token: 'test-token', user: { username: 'admin' } } };
      axios.post.mockResolvedValue(mockResponse);

      const result = await authAPI.login('admin', 'password');

      expect(axios.post).toHaveBeenCalledWith('/api/v1/auth/login', {
        username: 'admin',
        password: 'password',
      });
      expect(result).toEqual(mockResponse);
    });
  });
});
