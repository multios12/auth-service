import { createTheme, ThemeProvider } from '@mui/material/styles';
import './App.css';
import Login from './views/Login';

export default function App() {
  const theme = createTheme({ palette: { mode: 'dark' } })

  return (
    <ThemeProvider theme={theme}>
      <Login />
    </ThemeProvider>
  );
}
