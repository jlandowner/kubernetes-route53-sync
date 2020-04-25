FROM golang:1.14-alpine as build

RUN apk add --update --no-cache build-base git

# Create a "nobody" non-root user for the next image by crafting an /etc/passwd
# file that the next image can copy in. This is necessary since the next image
# is based on scratch, which doesn't have adduser, cat, echo, or even sh.
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

WORKDIR /go/src/kubernetes-route53-sync



COPY . . 
RUN CGO_ENABLED=0 go build -o /go/bin/kubernetes-route53-sync ./main.go

FROM alpine:3.10

# Copy the certs from the builder stage
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the /etc_passwd file we created in the builder stage into /etc/passwd in
# the target stage. This creates a new non-root user as a security best
# practice.
COPY --from=0 /etc_passwd /etc/passwd

COPY --from=build /go/bin/kubernetes-route53-sync /bin/kubernetes-route53-sync

USER nobody

ENTRYPOINT ["/bin/kubernetes-route53-sync"]
