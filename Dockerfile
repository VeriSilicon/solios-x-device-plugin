FROM debian:stable-slim

COPY solios-x-device-plugin /bin/
CMD ["solios-x-device-plugin"]