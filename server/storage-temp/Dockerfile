FROM openstf/stf:latest

USER root
RUN mkdir data && chown stf:stf data
USER stf
VOLUME ["data"]
