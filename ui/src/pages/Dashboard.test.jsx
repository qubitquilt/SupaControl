import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import Dashboard from './Dashboard';
import * as api from '../api';

// Mock the API module
vi.mock('../api', () => ({
  instancesAPI: {
    list: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
  },
}));

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe('Dashboard Component', () => {
  const mockOnLogout = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    // Set default successful mock for all tests
    api.instancesAPI.list.mockResolvedValue({ data: { instances: [] } });
  });

  const renderDashboard = () => {
    return render(
      <BrowserRouter>
        <Dashboard onLogout={mockOnLogout} />
      </BrowserRouter>
    );
  };

  describe('Initial Loading', () => {
    it('should render loading state initially', () => {
      api.instancesAPI.list.mockImplementation(() => new Promise(() => {}));
      renderDashboard();
      expect(screen.getByText('Loading instances...')).toBeInTheDocument();
    });

    it('should display header with title and buttons', async () => {
      renderDashboard();

      await waitFor(() => {
        expect(screen.queryByText('Loading instances...')).not.toBeInTheDocument();
      }, { timeout: 10000 });

      expect(screen.getByText('SupaControl')).toBeInTheDocument();
      expect(screen.getByText('Manage your Supabase instances')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /settings/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /logout/i })).toBeInTheDocument();
    });
  });

  describe('Instance Loading and Display', () => {
    it('should display empty state when no instances exist', async () => {
      renderDashboard();

      await waitFor(() => {
        expect(screen.queryByText('Loading instances...')).not.toBeInTheDocument();
      });

      expect(screen.getByText(/no instances yet/i)).toBeInTheDocument();
    });

    it('should display error message on load failure', async () => {
      api.instancesAPI.list.mockRejectedValue(new Error('Network error'));

      renderDashboard();

      await waitFor(() => {
        expect(screen.getByText('Failed to load instances')).toBeInTheDocument();
      });
    });
  });

  describe('Navigation', () => {
    it('should navigate to settings when clicking settings button', async () => {
      const user = userEvent.setup({ delay: null });
      renderDashboard();

      await waitFor(() => {
        expect(screen.queryByText('Loading instances...')).not.toBeInTheDocument();
      });

      const settingsButton = screen.getByRole('button', { name: /settings/i });
      await user.click(settingsButton);

      expect(mockNavigate).toHaveBeenCalledWith('/settings');
    });

    it('should call onLogout when clicking logout button', async () => {
      const user = userEvent.setup({ delay: null });
      renderDashboard();

      await waitFor(() => {
        expect(screen.queryByText('Loading instances...')).not.toBeInTheDocument();
      });

      const logoutButton = screen.getByRole('button', { name: /logout/i });
      await user.click(logoutButton);

      expect(mockOnLogout).toHaveBeenCalledTimes(1);
    });
  });
});
