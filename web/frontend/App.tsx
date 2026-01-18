import { h } from 'preact';
import { Router } from 'preact-router';
import { Create } from './pages/Create';
import { Read } from './pages/Read';
import { About } from './pages/About';
import { Layout } from './components/Layout';

function App() {
  return (
    <Layout>
      <Router>
        <Create path="/" />
        <Read path="/read/:id" />
        <About path="/about" />
      </Router>
    </Layout>
  );
}

export default App;
