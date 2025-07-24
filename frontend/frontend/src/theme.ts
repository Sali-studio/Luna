import { createTheme } from '@mui/material/styles';

const theme = createTheme({
  palette: {
    primary: {
      main: '#6750A4',
    },
    secondary: {
      main: '#625B71',
    },
    tertiary: {
      main: '#7D5260',
    },
    error: {
      main: '#B3261E',
    },
    background: {
      default: '#FFFBFE',
      paper: '#FFFBFE',
    },
    text: {
      primary: '#1C1B1F',
      secondary: '#49454F',
    },
  },
  typography: {
    fontFamily: 'Roboto, sans-serif',
    h1: {
      fontSize: '5.75rem',
      fontWeight: 400,
      letterSpacing: '-0.015625em',
    },
    h2: {
      fontSize: '3.625rem',
      fontWeight: 400,
      letterSpacing: '-0.00833em',
    },
    h3: {
      fontSize: '2.875rem',
      fontWeight: 400,
      letterSpacing: '0em',
    },
    h4: {
      fontSize: '2.0625rem',
      fontWeight: 400,
      letterSpacing: '0.00735em',
    },
    h5: {
      fontSize: '1.4375rem',
      fontWeight: 400,
      letterSpacing: '0em',
    },
    h6: {
      fontSize: '1.1875rem',
      fontWeight: 500,
      letterSpacing: '0.00156em',
    },
    subtitle1: {
      fontSize: '1rem',
      fontWeight: 400,
      letterSpacing: '0.00937em',
    },
    subtitle2: {
      fontSize: '0.875rem',
      fontWeight: 500,
      letterSpacing: '0.00714em',
    },
    body1: {
      fontSize: '1rem',
      fontWeight: 400,
      letterSpacing: '0.03125em',
    },
    body2: {
      fontSize: '0.875rem',
      fontWeight: 400,
      letterSpacing: '0.01786em',
    },
    button: {
      fontSize: '0.875rem',
      fontWeight: 500,
      letterSpacing: '0.01786em',
      textTransform: 'uppercase',
    },
    caption: {
      fontSize: '0.75rem',
      fontWeight: 400,
      letterSpacing: '0.03333em',
    },
    overline: {
      fontSize: '0.625rem',
      fontWeight: 500,
      letterSpacing: '0.1em',
      textTransform: 'uppercase',
    },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: '100px',
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: '12px',
        },
      },
    },
  },
});

export default theme;