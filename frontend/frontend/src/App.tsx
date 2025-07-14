import React, { useEffect, useState } from 'react';
import Box from '@mui/material/Box';
import Grid from '@mui/material/Unstable_Grid2'; // Import from Unstable_Grid2
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Avatar from '@mui/material/Avatar';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Sidebar from './components/Sidebar';
import SummaryCard from './components/SummaryCard';
import PeopleIcon from '@mui/icons-material/People';
import OnlinePredictionIcon from '@mui/icons-material/OnlinePrediction';
import DnsIcon from '@mui/icons-material/Dns';
import TerminalIcon from '@mui/icons-material/Terminal';
import { useAuth } from './contexts/AuthContext';
import DiscordIcon from './components/DiscordIcon';

interface DashboardData {
  totalUsers: number;
  onlineUsers: number;
  totalServers: number;
  commandsExecuted: number;
}

function AuthButton() {
  const { user, login, logout } = useAuth();
  const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);

  const handleMenu = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  if (user) {
    return (
      <div>
        <Button onClick={handleMenu} color="inherit" startIcon={<Avatar src={`https://cdn.discordapp.com/avatars/${user.id}/${user.avatar}.png`} sx={{ width: 32, height: 32 }} />}>
          {user.username}
        </Button>
        <Menu
          anchorEl={anchorEl}
          anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
          keepMounted
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          open={Boolean(anchorEl)}
          onClose={handleClose}
        >
          <MenuItem onClick={logout}>Logout</MenuItem>
        </Menu>
      </div>
    );
  }

  return (
    <Button 
      variant="contained" 
      onClick={login} 
      startIcon={<DiscordIcon />}
      sx={{ 
        backgroundColor: '#5865F2',
        color: 'white',
        '&:hover': {
          backgroundColor: '#4752C4',
        }
      }}
    >
      Login with Discord
    </Button>
  );
}

function App() {
  const [data, setData] = useState<DashboardData | null>(null);

  useEffect(() => {
    fetch('/api/dashboard')
      .then((res) => res.json())
      .then((data) => setData(data))
      .catch(err => console.error("Failed to fetch dashboard data:", err));
  }, []);

  return (
    <Box sx={{ display: 'flex', bgcolor: 'background.default' }}>
      <Sidebar />
      <Box component="main" sx={{ flexGrow: 1, p: { xs: 2, md: 4 } }}>
        <AppBar position="static" color="transparent" elevation={0} sx={{ mb: 4 }}>
          <Toolbar>
            <Typography variant="h4" noWrap component="div" sx={{ flexGrow: 1 }}>
              Dashboard
            </Typography>
            <AuthButton />
          </Toolbar>
        </AppBar>
        <Grid container spacing={3}>
          <Grid xs={12} sm={6} md={3}>
            <SummaryCard title="Total Users" value={data?.totalUsers.toLocaleString() || 'Loading...'} icon={<PeopleIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
          </Grid>
          <Grid xs={12} sm={6} md={3}>
            <SummaryCard title="Online Users" value={data?.onlineUsers.toLocaleString() || 'Loading...'} icon={<OnlinePredictionIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
          </Grid>
          <Grid xs={12} sm={6} md={3}>
            <SummaryCard title="Total Servers" value={data?.totalServers.toLocaleString() || 'Loading...'} icon={<DnsIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
          </Grid>
          <Grid xs={12} sm={6} md={3}>
            <SummaryCard title="Commands Executed" value={data?.commandsExecuted.toLocaleString() || 'Loading...'} icon={<TerminalIcon sx={{ fontSize: 32, color: 'primary.main' }} />} />
          </Grid>
        </Grid>
      </Box>
    </Box>
  );
}

export default App;