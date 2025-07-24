"use client";

import React from 'react';
import { Typography } from '@mui/material';
import DashboardLayout from '../components/DashboardLayout';

function Home() {
  return (
    <DashboardLayout>
      <Typography variant="h4" gutterBottom sx={{ mb: 4 }}>
        Dashboard Overview
      </Typography>

      <Typography variant="body1">
        This is the main dashboard content area. Cards will be added here later.
      </Typography>
    </DashboardLayout>
  );
}

export default Home;