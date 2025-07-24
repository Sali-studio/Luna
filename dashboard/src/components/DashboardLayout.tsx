"use client";

import React from 'react';
import { Box, AppBar, Toolbar, Typography, Button, IconButton, List, ListItem, ListItemButton, ListItemIcon, ListItemText } from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import DashboardIcon from '@mui/icons-material/Dashboard';
import SettingsIcon from '@mui/icons-material/Settings';
import Link from 'next/link';
import { useTheme } from '@mui/material/styles';

interface DashboardLayoutProps {
  children: React.ReactNode;
}

function DashboardLayout({ children }: DashboardLayoutProps) {
  const theme = useTheme();

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
          <Button color="inherit">Login</Button>
        </Toolbar>
      </AppBar>
      <Box sx={{ display: 'flex', flexGrow: 1, p: 2 }}>
        {/* サイドバー */}
        <Box sx={{
          width: 240,
          flexShrink: 0,
          mr: 2,
          p: 2,
          bgcolor: 'background.paper',
          borderRadius: '16px',
          boxShadow: theme.shadows[1],
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
          bgcolor: 'background.paper',
          borderRadius: '16px',
          boxShadow: theme.shadows[1],
        }}>
          {children}
        </Box>
      </Box>
    </Box>
  );
}

export default DashboardLayout;
