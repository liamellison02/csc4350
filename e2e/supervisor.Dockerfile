# combined supervisor + collector image for e2e; the supervisor image
# alone has no agent binary to exec. paths verified against docker inspect.
FROM otel/opentelemetry-collector-opampsupervisor:0.147.0 AS sup
FROM otel/opentelemetry-collector-contrib:0.147.0 AS col

FROM alpine:3.22
COPY --from=sup /usr/local/bin/opampsupervisor /opampsupervisor
COPY --from=col /otelcol-contrib /otelcol-contrib
RUN mkdir -p /var/lib/otelcol/supervisor
ENTRYPOINT ["/opampsupervisor"]
CMD ["--config", "/etc/supervisor.yaml"]
