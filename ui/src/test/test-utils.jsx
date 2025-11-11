import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';

// Custom render function that includes providers
export function renderWithRouter(ui, { route = '/', ...renderOptions } = {}) {
  window.history.pushState({}, 'Test page', route);

  return render(
    <BrowserRouter>
      {ui}
    </BrowserRouter>,
    renderOptions
  );
}

// Re-export everything from React Testing Library
export * from '@testing-library/react';
export { renderWithRouter as render };
