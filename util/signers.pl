#!/usr/bin/perl -w
use strict;
use File::Temp qw/tempfile/;
use Data::Dumper;

my $action = $ARGV[0] || '';

my $goalou = "";
my $tosign = "";
if( $action eq "sign" ) {
  $goalou = $ARGV[1];
  $tosign = $ARGV[2];
}
my $found = 0;

my $rawcerts = `security find-certificate -a -p -Z`;
$rawcerts .= "SHA-1 hash: 000";

my %certs;
my $cert = "";
my $hash = "";
for my $line ( split( '\n', $rawcerts ) ) {
    if( $line =~ m/^SHA-1 hash: ([0-9A-F]+)$/ ) {
        my $linehash = $1;
        if( $hash ) {
            $certs{ $hash } = $cert;
        }
        $cert = "";
        $hash = $linehash;
        next;
    }
    $cert .= "$line\n";
}

my $signers = `security find-identity -v -p codesigning`;
my $type = "Mac Developer";
for my $line ( split( '\n', $signers ) ) {
    if( $line =~ m/[0-9]+\) ([A-Z0-9]+) "$type: (.+)"$/ ) {
        my $linehash = $1;
        my $linename = $2;
        my $cert = $certs{ $linehash };
        decode_cert( $cert );
    }
}

if( $action eq 'sign' ) {
  if( !$found ) {
    print STDERR "Could not find Mac Developer cert for developer OU $goalou\n";
    exit 1;
  }
  `codesign -fs "$found" "$tosign"`;
  print `codesign -d -r- "$tosign"`;
}
exit 0;

sub decode_cert {
    my $cert = shift;
    
    my ( $fh, $fname ) = tempfile( UNLINK => 1 );
    print $fh $cert;
    close( $fh );
    
    my $text = `openssl x509 -text -in $fname -noout`;
    for my $line ( split( '\n', $text ) ) {
      if( $line =~ m/Subject:.+CN=$type: (.+), OU=([A-Z0-9]+),/ ) {
        my $name = $1;
        my $ou = $2;
        if( !$action ) { 
          print "Name: $name, OU: $ou\n";
        }
        else {
          if( $goalou eq $ou ) {
            $found = "$type: $name";
          }
        }
      }
    }
}