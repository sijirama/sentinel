import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import StatusComponent from './components/StatusPage';
const queryClient = new QueryClient();

function App() {
    return (
        <QueryClientProvider client={queryClient}>
            <StatusComponent />
        </QueryClientProvider>
    )
}

export default App
