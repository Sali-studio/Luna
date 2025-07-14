import React from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Typography from '@mui/material/Typography';
import DashboardIcon from '@mui/icons-material/Dashboard';
import SettingsIcon from '@mui/icons-material/Settings';
import AnalyticsIcon from '@mui/icons-material/Analytics';

const Sidebar = () => {
  const theme = useTheme();
  const [selectedIndex, setSelectedIndex] = React.useState(0);

  const handleListItemClick = (index: number) => {
    setSelectedIndex(index);
  };

  return (
    <Box sx={{ 
      p: 2,
      height: '100vh',
      position: 'sticky',
      top: 0,
    }}>
      <Box sx={{ 
        bgcolor: 'background.paper',
        p: 2,
        borderRadius: '16px',
        height: '100%',
        display: 'flex',
        flexDirection: 'column'
      }}>
        <Typography variant="h6" color="text.primary" sx={{ p: 2, fontWeight: 700 }}>
          Luna Dashboard
        </Typography>
        <List>
          <ListItem disablePadding>
            <ListItemButton
              selected={selectedIndex === 0}
              onClick={() => handleListItemClick(0)}
              sx={{
                borderRadius: '12px',
                py: 1.5,
                px: 2,
                mb: 1,
                color: 'text.secondary',
                '&.Mui-selected': {
                  backgroundColor: theme.palette.primary.main + '20', // primary with low opacity
                  color: 'primary.main',
                  '& .MuiListItemIcon-root': {
                    color: 'primary.main',
                  },
                },
                '&:hover': {
                  backgroundColor: theme.palette.action.hover,
                },
              }}
            >
              <ListItemIcon sx={{ color: 'inherit', minWidth: 40 }}>
                <DashboardIcon />
              </ListItemIcon>
              <ListItemText primary="Dashboard" />
            </ListItemButton>
          </ListItem>
          {/* ... other list items */}
        </List>
      </Box>
    </Box>
  );
};

export default Sidebar;