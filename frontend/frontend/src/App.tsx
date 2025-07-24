import React from 'react';
import { Box, AppBar, Toolbar, Typography, Button } from '@mui/material';

function App() {
  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Luna Dashboard
          </Typography>
          <Button color="inherit">Login</Button>
        </Toolbar>
      </AppBar>
      <Box sx={{ p: 3 }}>
        <Typography variant="h4" gutterBottom>
          Welcome to the new Dashboard!
        </Typography>
        <Typography variant="body1">
          This is a fresh start for your Luna Bot Web Dashboard.
        </Typography>
      </Box>
    </Box>
  );
}

export default App;
