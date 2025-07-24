"use client";

import "./globals.css";
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import theme from '../theme'; // theme.tsのパスを修正
import { AuthProvider } from '../contexts/AuthContext'; // AuthProviderをインポート
import AnimatedBlobs from '../components/AnimatedBlobs'; // AnimatedBlobsをインポート

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <AnimatedBlobs /> {/* AnimatedBlobsコンポーネントを配置 */}
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <AuthProvider> {/* AuthProviderでchildrenをラップ */}
            {children}
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
