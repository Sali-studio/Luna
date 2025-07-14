import React, { useEffect, useState } from 'react';
import { ThemeProvider, createTheme, responsiveFontSizes } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import Box from '@mui/material/Box';
import Grid from '@mui/material/Grid';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Sidebar from './components/Sidebar';
import SummaryCard from './components/SummaryCard';
import CommandUsageChart from './components/CommandUsageChart';
import PeopleIcon from '@mui/icons-material/People';
import OnlinePredictionIcon from '@mui/icons-material/OnlinePrediction';
import DnsIcon from '@mui/icons-material/Dns';
import TerminalIcon from '@mui/icons-material/Terminal';

let m3Theme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#A8C7FA',
    },
    background: {
      default: '#1F1F23',
      paper: '#2A2A2E',
    },
    text: {
      primary: '#E3E3E3',
      secondary: '#C7C7C7',
    },
  },
  shape: {
    borderRadius: 20,
  },
  typography: {
    fontFamily: '"Plus Jakarta Sans", sans-serif',
    h4: { fontWeight: 700 },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundColor: '#2A2A2E',
          backgroundImage: 'none',
          boxShadow: 'none',
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: 'transparent',
        },
      },
    },
  },
});
m3Theme = responsiveFontSizes(m3Theme);

interface DashboardData {
  totalUsers: number;
  onlineUsers: number;
  totalServers: number;
  commandsExecuted: number;
  commandUsage: { name: string; count: number }[];
}

function App() {
  const [data, setData] = useState<DashboardData | null>(null);

  useEffect(() => {
    fetch('/api/dashboard')
      .then((res) => res.json())
      .then((data) => setData(data));
  }, []);

  return (
    <ThemeProvider theme={m3Theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', bgcolor: 'background.default' }}>
        <Sidebar />
        <Box component="main" sx={{ flexGrow: 1, p: { xs: 2, md: 4 } }}>
          <AppBar position="static" color="transparent" elevation={0} sx={{ mb: 4 }}>
            <Toolbar>
              <Typography variant="h4" noWrap component="div" sx={{ flexGrow: 1 }}>
                Dashboard
              </Typography>
            </Toolbar>
          </AppBar>
          <Grid container spacing={3}>
            <Grid item xs={12} lg={8}>
              <CommandUsageChart data={data?.commandUsage} />
            </Grid>
            <Grid item xs={12} lg={4}>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6} lg={12}>
                  <SummaryCard title="Total Users" value={data?.totalUsers.toLocaleString() || 'Loading...'} icon={<PeopleIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
                </Grid>
                <Grid item xs={12} sm={6} lg={12}>
                  <SummaryCard title="Online Users" value={data?.onlineUsers.toLocaleString() || 'Loading...'} icon={<OnlinePredictionIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
                </Grid>
                <Grid item xs={12} sm={6} lg={12}>
                  <SummaryCard title="Total Servers" value={data?.totalServers.toLocaleString() || 'Loading...'} icon={<DnsIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
                </Grid>
                <Grid item xs={12} sm={6} lg={12}>
                  <SummaryCard title="Commands Executed" value={data?.commandsExecuted.toLocaleString() || 'Loading...'} icon={<TerminalIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Box>
      </Box>
    </ThemeProvider>
  );
}

export default App;