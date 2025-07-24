import { createTheme } from '@mui/material/styles';

const theme = createTheme({
  palette: {
    mode: 'dark', // ダークモードを有効化
    primary: {
      main: '#81D4FA', // Lunaのイメージカラーである水色（ダークモード向けに調整）
    },
    secondary: {
      main: '#B0BEC5',
    },
    tertiary: {
      main: '#FFAB91',
    },
    error: {
      main: '#FF8A80',
    },
    background: {
      default: '#121212', // ダークな背景色
      paper: '#1E1E1E', // カードなどの背景色
    },
    text: {
      primary: '#E0E0E0',
      secondary: '#BDBDBD',
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
          backgroundColor: 'rgba(40, 39, 41, 0.3)', // 透明度を0.3に
          boxShadow: '0px 4px 20px rgba(0, 0, 0, 0.3)', // 影を強調
          backdropFilter: 'blur(8px)', // すりガラス効果を追加
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          borderRadius: '16px',
          boxShadow: 'none', // 影をなくす
          backgroundColor: 'rgba(33, 31, 33, 0.8)', // #211f21 に変更し、透明度を維持
          backdropFilter: 'blur(12px)', // ぼかしはそのまま
          border: '1px solid rgba(255, 255, 255, 0.1)', // 微細な境界線（ダークモード向け）
        },
      },
    },
  },
});

export default theme;
