"use client";

import React from 'react';
import { Box, Typography } from '@mui/material';
import DashboardLayout from '../../components/DashboardLayout';

function SettingsPage() {
  return (
    <DashboardLayout>
      <Box sx={{ p: 3 }}>
        <Typography variant="h4" gutterBottom>
          Settings
        </Typography>
        <Typography variant="body1">
          This is the settings page. You can configure various bot settings here.
        </Typography>
      </Box>
    </DashboardLayout>
  );
}

export default SettingsPage;