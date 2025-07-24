"use client";

import React from 'react';
import { Box, AppBar, Toolbar, Typography, Button, IconButton, List, ListItem, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import DashboardIcon from '@mui/icons-material/Dashboard';
import SettingsIcon from '@mui/icons-material/Settings';
import Link from 'next/link';
import { useTheme } from '@mui/material/styles';
import { useAuth } from '../contexts/AuthContext'; // useAuthをインポート

interface DashboardLayoutProps {
  children: React.ReactNode;
}

function DashboardLayout({ children }: DashboardLayoutProps) {
  const theme = useTheme();
  const { user, loading, login, logout } = useAuth(); // useAuthを使用

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <AppBar position="static" sx={{ m: 2, borderRadius: '16px', width: 'calc(100% - 32px)' }}>
        <Toolbar>
          <IconButton
            size="large"
            edge="start"
            color="inherit"
            aria-label="menu"
            sx={{ mr: 2 }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Luna Dashboard
          </Typography>
          {loading ? (
            <Typography color="inherit">Loading...</Typography>
          ) : user ? (
            <Button color="inherit" onClick={logout}>
              Logout ({user.username})
            </Button>
          ) : (
            <Button color="inherit" onClick={login}>
              Login
            </Button>
          )}
        </Toolbar>
      </AppBar>
      <Box sx={{ display: 'flex', flexGrow: 1, p: 2 }}>
        {/* サイドバー */}
        <Box sx={{
          width: 240,
          flexShrink: 0,
          mr: 2,
          p: 2,
          bgcolor: 'rgba(28, 27, 29, 0.85)', // #1c1b1d に変更し、透明度を維持
          borderRadius: '16px',
          boxShadow: '0px 4px 20px rgba(0, 0, 0, 0.3)', // 影を強調
          backdropFilter: 'blur(10px)', // すりガラス効果
        }}>
          <List>
            <ListItem disablePadding>
              <ListItemButton component={Link} href="/">
                <ListItemIcon>
                  <DashboardIcon />
                </ListItemIcon>
                <ListItemText primary="Dashboard" />
              </ListItemButton>
            </ListItem>
            <ListItem disablePadding>
              <ListItemButton component={Link} href="/settings">
                <ListItemIcon>
                  <SettingsIcon />
                </ListItemIcon>
                <ListItemText primary="Settings" />
              </ListItemButton>
            </ListItem>
          </List>
        </Box>

        {/* メインコンテンツエリア */}
        <Box sx={{
          flexGrow: 1,
          p: 2,
          bgcolor: 'rgba(28, 27, 29, 0.85)', // #1c1b1d に変更し、透明度を維持
          borderRadius: '16px',
          boxShadow: '0px 4px 20px rgba(0, 0, 0, 0.3)', // 影を強調
          backdropFilter: 'blur(10px)', // すりガラス効果
        }}>
          {children}
        </Box>
      </Box>
    </Box>
  );
}

export default DashboardLayout;
