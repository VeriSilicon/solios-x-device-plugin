FROM alpine:latest

COPY solios-x-device-plugin /root/

CMD ["/root/solios-x-device-plugin"]
