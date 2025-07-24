"use client";

import React from 'react';
import { Typography } from '@mui/material';
import DashboardLayout from '../components/DashboardLayout';

function Home() {
  return (
    <DashboardLayout>
      <Typography variant="h4" gutterBottom>
        Welcome to Luna Dashboard!
      </Typography>
      <Typography variant="body1">
        This is the new web dashboard for your Luna Bot.
      </Typography>
    </DashboardLayout>
  );
}

export default Home;