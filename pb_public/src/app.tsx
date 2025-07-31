import { Router, Route } from 'preact-router';
import { Header } from './components/header';
import { Dashboard } from './pages/dashboard';
import { ServiceDetail } from './pages/service-detail';
import { CreateService } from './pages/create-service';

export function App() {
  return (
    <div class="min-h-screen flex flex-col">
      <Header />
      <main class="container mx-auto px-4 py-8 flex-grow">
        <Router>
          <Route path="/" component={Dashboard} />
          <Route path="/services/new" component={CreateService} />
          <Route path="/services/:id" component={ServiceDetail} />
        </Router>
      </main>
      <footer class="footer footer-center p-4 bg-base-300 text-base-content">
        <div>
          <p>Pockestrator - PocketBase Manager</p>
        </div>
      </footer>
    </div>
  );
}