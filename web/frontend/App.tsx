import { h } from 'preact';
import { Router } from 'preact-router';
import { Create } from './pages/Create';
import { Read } from './pages/Read';
import { About } from './pages/About';
import { Layout } from './components/Layout';
import { useConfig } from './hooks/useConfig';
import { useTheme } from './hooks/useTheme';

function App() {
  const config = useConfig();
  const { toggleTheme } = useTheme(config.default_theme);

  return (
    <Layout onToggleTheme={toggleTheme}>
      <Router>
        <Create path="/" />
        <Read path="/read/:id" />
        <About path="/about" />
      </Router>
    </Layout>
  );
}

export default App;
