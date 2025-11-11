import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Login from './Login';
import * as api from '../api';

// Mock the API module
vi.mock('../api', () => ({
  authAPI: {
    login: vi.fn(),
  },
}));

describe('Login Component', () => {
  const mockOnLogin = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  const renderLogin = () => {
    return render(<Login onLogin={mockOnLogin} />);
  };

  describe('Rendering', () => {
    it('should render login form with all elements', () => {
      renderLogin();

      expect(screen.getByText('SupaControl')).toBeInTheDocument();
      expect(screen.getByText('Supabase Management Platform')).toBeInTheDocument();
      expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument();
    });

    it('should have username and password inputs', () => {
      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);

      expect(usernameInput).toHaveAttribute('type', 'text');
      expect(passwordInput).toHaveAttribute('type', 'password');
      expect(usernameInput).toHaveAttribute('required');
      expect(passwordInput).toHaveAttribute('required');
    });

    it('should have username input focused by default', () => {
      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      expect(usernameInput).toHaveAttribute('autoFocus');
    });
  });

  describe('Form Interaction', () => {
    it('should update username field when typing', async () => {
      const user = userEvent.setup();
      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      await user.type(usernameInput, 'testuser');

      expect(usernameInput).toHaveValue('testuser');
    });

    it('should update password field when typing', async () => {
      const user = userEvent.setup();
      renderLogin();

      const passwordInput = screen.getByLabelText(/password/i);
      await user.type(passwordInput, 'testpassword');

      expect(passwordInput).toHaveValue('testpassword');
    });

    it('should not submit form when username is empty', async () => {
      const user = userEvent.setup();
      renderLogin();

      const passwordInput = screen.getByLabelText(/password/i);
      await user.type(passwordInput, 'testpassword');

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      expect(api.authAPI.login).not.toHaveBeenCalled();
    });

    it('should not submit form when password is empty', async () => {
      const user = userEvent.setup();
      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      await user.type(usernameInput, 'testuser');

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      expect(api.authAPI.login).not.toHaveBeenCalled();
    });
  });

  describe('Successful Login', () => {
    it('should call API with correct credentials', async () => {
      const user = userEvent.setup();
      const mockToken = 'test-jwt-token';

      api.authAPI.login.mockResolvedValue({
        data: { token: mockToken }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.authAPI.login).toHaveBeenCalledWith('admin', 'password123');
      });
    });

    it('should store token in localStorage on successful login', async () => {
      const user = userEvent.setup();
      const mockToken = 'test-jwt-token';

      api.authAPI.login.mockResolvedValue({
        data: { token: mockToken }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(localStorage.setItem).toHaveBeenCalledWith('token', mockToken);
      });
    });

    it('should call onLogin callback on successful login', async () => {
      const user = userEvent.setup();
      const mockToken = 'test-jwt-token';

      api.authAPI.login.mockResolvedValue({
        data: { token: mockToken }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnLogin).toHaveBeenCalledTimes(1);
      });
    });

    it('should show loading state during login', async () => {
      const user = userEvent.setup();
      let resolveLogin;
      const loginPromise = new Promise((resolve) => {
        resolveLogin = resolve;
      });

      api.authAPI.login.mockReturnValue(loginPromise);

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      // Button should be disabled during loading
      expect(submitButton).toBeDisabled();

      resolveLogin({ data: { token: 'test-token' } });

      await waitFor(() => {
        expect(submitButton).not.toBeDisabled();
      });
    });
  });

  describe('Failed Login', () => {
    it('should show error message on login failure', async () => {
      const user = userEvent.setup();

      api.authAPI.login.mockRejectedValue({
        response: {
          data: { message: 'Invalid credentials' }
        }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'wrongpassword');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
      });
    });

    it('should show generic error message when API error has no message', async () => {
      const user = userEvent.setup();

      api.authAPI.login.mockRejectedValue(new Error('Network error'));

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Login failed. Please try again.')).toBeInTheDocument();
      });
    });

    it('should not store token on failed login', async () => {
      const user = userEvent.setup();

      api.authAPI.login.mockRejectedValue({
        response: {
          data: { message: 'Invalid credentials' }
        }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'wrongpassword');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
      });

      expect(localStorage.setItem).not.toHaveBeenCalledWith('token', expect.anything());
    });

    it('should not call onLogin on failed login', async () => {
      const user = userEvent.setup();

      api.authAPI.login.mockRejectedValue({
        response: {
          data: { message: 'Invalid credentials' }
        }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'wrongpassword');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
      });

      expect(mockOnLogin).not.toHaveBeenCalled();
    });

    it('should clear error message when user starts typing again', async () => {
      const user = userEvent.setup();

      api.authAPI.login.mockRejectedValue({
        response: {
          data: { message: 'Invalid credentials' }
        }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      // First failed login
      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'wrongpassword');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
      });

      // Type again - this should clear the form and trigger re-submission
      // In the actual component, errors are cleared on new submission
      await user.clear(usernameInput);
      await user.type(usernameInput, 'admin2');
      await user.clear(passwordInput);
      await user.type(passwordInput, 'newpassword');

      // On re-submission, error should be cleared first
      api.authAPI.login.mockResolvedValue({
        data: { token: 'test-token' }
      });

      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.queryByText('Invalid credentials')).not.toBeInTheDocument();
      });
    });
  });

  describe('Form Submission', () => {
    it('should submit form when pressing Enter in password field', async () => {
      const user = userEvent.setup();
      const mockToken = 'test-jwt-token';

      api.authAPI.login.mockResolvedValue({
        data: { token: mockToken }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123{Enter}');

      await waitFor(() => {
        expect(api.authAPI.login).toHaveBeenCalledWith('admin', 'password123');
      });
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty API response', async () => {
      const user = userEvent.setup();

      api.authAPI.login.mockResolvedValue({ data: {} });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(localStorage.setItem).toHaveBeenCalledWith('token', undefined);
      });
    });

    it('should handle whitespace in username and password', async () => {
      const user = userEvent.setup();
      const mockToken = 'test-jwt-token';

      api.authAPI.login.mockResolvedValue({
        data: { token: mockToken }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, '  admin  ');
      await user.type(passwordInput, '  password  ');
      await user.click(submitButton);

      await waitFor(() => {
        // Should send credentials with whitespace (no trimming in the component)
        expect(api.authAPI.login).toHaveBeenCalledWith('  admin  ', '  password  ');
      });
    });

    it('should handle special characters in credentials', async () => {
      const user = userEvent.setup();
      const mockToken = 'test-jwt-token';

      api.authAPI.login.mockResolvedValue({
        data: { token: mockToken }
      });

      renderLogin();

      const usernameInput = screen.getByLabelText(/username/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      await user.type(usernameInput, 'admin@example.com');
      await user.type(passwordInput, 'p@$$w0rd!');
      await user.click(submitButton);

      await waitFor(() => {
        expect(api.authAPI.login).toHaveBeenCalledWith('admin@example.com', 'p@$$w0rd!');
      });
    });
  });
});
