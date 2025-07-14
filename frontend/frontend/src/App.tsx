import React from 'react';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
  },
});

function App() {
  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />
      <Box sx={{ display: 'flex' }}>
        {/* TODO: Add Sidebar */}
        <Box
          component="main"
          sx={{ flexGrow: 1, bgcolor: 'background.default', p: 3 }}
        >
          <Typography variant="h4" gutterBottom>
            Welcome to the Luna Dashboard
          </Typography>
          <Typography paragraph>
            This is where the main content will be displayed.
          </Typography>
        </Box>
      </Box>
    </ThemeProvider>
  );
}

export default App;
