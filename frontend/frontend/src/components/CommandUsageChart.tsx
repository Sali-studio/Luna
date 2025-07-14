import React from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';

interface CommandUsageChartProps {
  data: { name: string; count: number }[] | undefined;
}

const CommandUsageChart: React.FC<CommandUsageChartProps> = ({ data }) => {
  const theme = useTheme();

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          Command Usage Trend
        </Typography>
        <ResponsiveContainer width="100%" height={400}>
          <AreaChart data={data} margin={{ top: 20, right: 30, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="colorUv" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={theme.palette.primary.main} stopOpacity={0.5}/>
                <stop offset="95%" stopColor={theme.palette.primary.main} stopOpacity={0}/>
              </linearGradient>
            </defs>
            <CartesianGrid stroke={theme.palette.background.default} strokeDasharray="5 5" />
            <XAxis dataKey="name" stroke={theme.palette.text.secondary} tick={{ fill: theme.palette.text.secondary }} />
            <YAxis stroke={theme.palette.text.secondary} tick={{ fill: theme.palette.text.secondary }} />
            <Tooltip
              contentStyle={{ 
                backgroundColor: theme.palette.background.paper,
                borderColor: theme.palette.divider,
                borderRadius: '12px'
              }}
            />
            <Area type="monotone" dataKey="count" stroke={theme.palette.primary.main} strokeWidth={2} fillOpacity={1} fill="url(#colorUv)" />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
};

export default CommandUsageChart;