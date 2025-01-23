# Stage 1: Build the application
FROM node:18 AS build

# Set the working directory
WORKDIR /usr/src/app

# Copy package.json and yarn.lock
COPY package.json yarn.lock ./

# Install dependencies
RUN yarn install

# Copy the rest of the application code
COPY . .

# Build the application
RUN yarn build

# Stage 2: Serve the application with Nginx
FROM nginx:alpine

# Define build arguments
ARG VERSION
ARG BUILD_DATE
ARG COMMIT_SHA

# Copy the built files to the Nginx HTML directory
COPY --from=build /usr/src/app/dist /usr/share/nginx/html

# Copy custom Nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Add OCI annotations as labels
LABEL org.opencontainers.image.title="Polaris Service"
LABEL org.opencontainers.image.description="Polaris is the central hub and orchestration layer for the Asterism project. It provides a unified interface to manage, monitor, and interact with distributed microservices, enabling seamless integration of services and microfrontends. Polaris facilitates service discovery, communication, and coordination while ensuring an intuitive user experience through its modular and scalable architecture."
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.url="https://github.com/jlfowle/asterism"
LABEL org.opencontainers.image.source="https://github.com/jlfowle/asterism"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="Jas Fowler <jasph.fowler@gmail.com>"
LABEL org.opencontainers.image.documentation="https://github.com/jlfowle/asterism/docs"
LABEL org.opencontainers.image.vendor="Asterism Project"
LABEL org.opencontainers.image.ref.name="${VERSION}"

# Expose the port Nginx runs on
EXPOSE 80

# Command to run Nginx
CMD ["nginx", "-g", "daemon off;"]
