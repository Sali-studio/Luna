import { createTheme, responsiveFontSizes } from '@mui/material/styles';

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

export default m3Theme;
