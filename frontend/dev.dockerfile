FROM node:lts-alpine
 
WORKDIR /app
COPY . .
 
RUN npm ci
 
EXPOSE 4321
 
# Use the dev command specified in your package.json
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

