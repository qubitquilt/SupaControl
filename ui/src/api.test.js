import { describe, it, expect, beforeEach, vi } from 'vitest';

// Mock axios before importing api
vi.mock('axios', () => {
  return {
    default: {
      create: vi.fn(() => ({
        interceptors: {
          request: { use: vi.fn() },
          response: { use: vi.fn() },
        },
        get: vi.fn(),
        post: vi.fn(),
        delete: vi.fn(),
      })),
      post: vi.fn(),
    },
  };
});

describe('API Configuration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('should export authAPI', async () => {
    const { authAPI } = await import('./api');
    expect(authAPI).toBeDefined();
    expect(authAPI.login).toBeDefined();
  });

  it('should export instancesAPI', async () => {
    const { instancesAPI } = await import('./api');
    expect(instancesAPI).toBeDefined();
    expect(instancesAPI.create).toBeDefined();
    expect(instancesAPI.list).toBeDefined();
  });
});
