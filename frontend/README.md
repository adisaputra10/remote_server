# GoTeleport Frontend

Modern Vue.js frontend for GoTeleport remote server management system.

## Features

- 📊 **Dashboard** - Real-time statistics and overview
- 📝 **Command Logs** - View and filter executed commands
- 📋 **Access Logs** - Monitor user access and activities  
- 🔌 **Sessions** - Manage active connections
- 🎨 **Modern UI** - Clean and responsive design with Element Plus
- 🔄 **Real-time Updates** - Live data refreshing
- 📤 **Export** - CSV export for logs

## Tech Stack

- **Vue 3** - Progressive JavaScript framework
- **Element Plus** - Vue 3 UI library
- **Vite** - Fast build tool
- **Axios** - HTTP client
- **Vue Router** - Single page application routing

## Quick Start

### Prerequisites

- Node.js 16+ 
- npm or yarn
- GoTeleport backend server running on port 8080

### Development

1. **Install dependencies:**
   ```bash
   npm install
   ```

2. **Start development server:**
   ```bash
   npm run dev
   ```
   
   Or use the Windows batch file:
   ```cmd
   start-frontend.bat
   ```

3. **Open browser:**
   ```
   http://localhost:3000
   ```

### Production Build

1. **Build for production:**
   ```bash
   npm run build
   ```
   
   Or use the Windows batch file:
   ```cmd
   build-frontend.bat
   ```

2. **Preview production build:**
   ```bash
   npm run preview
   ```

## Configuration

### API Proxy

Development server automatically proxies API requests to the backend:

```javascript
// vite.config.js
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
    secure: false
  }
}
```

### Environment Variables

Create `.env.local` for custom configuration:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_APP_TITLE=GoTeleport Dashboard
```

## Project Structure

```
frontend/
├── public/                 # Static assets
├── src/
│   ├── components/        # Reusable Vue components
│   ├── views/            # Page components
│   ├── services/         # API services
│   ├── router/           # Vue Router configuration
│   ├── App.vue           # Root component
│   └── main.js           # Application entry point
├── package.json          # Dependencies and scripts
├── vite.config.js        # Vite configuration
└── README.md            # This file
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run serve` - Serve production build (after build)

## API Integration

The frontend communicates with the GoTeleport backend via REST API:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/stats` | GET | System statistics |
| `/api/logs` | GET | Command logs with filtering |
| `/api/access-logs` | GET | Access logs with filtering |
| `/api/sessions` | GET | Active sessions |

### API Response Format

All API endpoints return data in this format:
```json
{
  "logs": [...],
  "total": 100
}
```

## Features Detail

### Dashboard
- Real-time connection statistics
- Recent command and access logs
- Quick navigation to detailed views

### Command Logs
- Filter by session, client, agent, status
- Real-time updates every 30 seconds
- CSV export functionality
- Pagination support

### Access Logs
- Monitor user login/logout activities
- Filter by client, agent, username, action
- Export capabilities

### Sessions
- View active connections
- Session management
- Real-time status updates

## Development

### Adding New Features

1. Create new Vue component in `src/views/`
2. Add route in `src/router/index.js`
3. Add API method in `src/services/api.js`
4. Update navigation in `App.vue`

### Styling

Uses Element Plus components with custom CSS:
- Consistent color scheme
- Responsive design
- Professional appearance

## Deployment

### With Backend Server

The built frontend can be served by the Go backend server by copying the `dist/` folder to the server's static directory.

### Standalone Deployment

Deploy the built files to any static hosting service:
- Netlify
- Vercel
- GitHub Pages
- Apache/Nginx

## Browser Support

- Chrome 87+
- Firefox 78+
- Safari 14+
- Edge 88+

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

This project is part of GoTeleport and follows the same license terms.
