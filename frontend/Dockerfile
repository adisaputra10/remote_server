# Dockerfile for Vue.js frontend
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Production image
FROM nginx:alpine

# Install wget for healthcheck, remove default nginx config and setup nginx user
RUN apk add --no-cache wget 
# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built application and set permissions
COPY --from=build /app/dist /usr/share/nginx/html
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chmod -R 755 /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
