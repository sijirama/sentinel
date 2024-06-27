FROM oven/bun


WORKDIR /app
COPY . .

RUN bun install

EXPOSE 5173
CMD ["bun", "dev" , "--", "--host", "0.0.0.0"]
