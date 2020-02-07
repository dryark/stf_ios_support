#!/usr/bin/perl -w
use strict;
use Data::Dumper;
use MIME::Base64;
use JSON::PP;

my $checkapp = '';
my $action = $ARGV[0] || '';
if( $action eq 'permok' || $action eq 'ensureperm' ) {
    $checkapp = $ARGV[1] || "/Applications/STF Coordinator.app";
}
elsif( $action eq 'dump' ) {}
else {
  print "ALF Firewall / Network Permissions Tool\n
  Usage:
    ./alf2.pl [action] [args]
  Actions:
    dump - dump contents of permissions
    permok [path to app or binary] - check if a app or binary has permission
    ensureperm [path to app or binary] - ensure an app or binary has permission\n";
}

my $i = 1;
my $app;
my %apps;
for my $line ( `/usr/libexec/ApplicationFirewall/socketfilterfw --listapps` ) {
    if( $line =~ m/^$i\s+:\s+(.+?)\s*$/ ) {
        $app = $1;
        $i++;
    }
        
    if( $line =~ m/Allow incoming connections/ ) {
        if( !$checkapp || $checkapp eq $app ) {
            my $csreq = get_cs_req( $i - 2 );
            chomp $csreq;
            my $info = {
                csreq => $csreq,
                valid => 1
            };
            if( $csreq =~ m/cdhash H"(.+?)"/ ) {
                my $sha1 = uc($1);
                my $cdhash = get_cdhash( $app );
                $info->{ valid } = ( $sha1 eq $cdhash ) ? 1 : 0;
            }
            $apps{ $app } = $info;
        }
        else {
            $apps{ $app } = {};
        }
    }
}

if( $action eq 'permok' ) {
    my $info = $apps{ $checkapp };
    print $info->{valid} ? 'yes' : 'no';
}
elsif( $action eq 'ensureperm' ) {
    # TODO
}
elsif( $action eq 'dump' ) {
    print JSON::PP->new->ascii->pretty->encode( \%apps );
}

sub get_cs_req {
    my $i = shift;
    my $grab = 0;
    my $b64;
    for my $line ( `plutil -extract applications.$i.reqdata xml1 -o - /Library/Preferences/com.apple.alf.plist` ) {
        if( $line =~ m/<data>/ ) {
            $grab = 1;
            next;
        }
        if( $grab ) {
            $b64 = $line;
            last;
        }
    }
    my $raw = decode_base64( $b64 );
    open( my $fh, ">csreq.bin" );
    binmode( $fh );
    print $fh $raw;
    my $csreq = `csreq -r csreq.bin -t`;
    close( $fh );
    unlink( "csreq.bin" );
    return $csreq;
}

sub get_cdhash {
    my $path = shift;
    for my $line ( `codesign -dvvv "$path" 2>&1` ) {
        if( $line =~ m/^CDHash=(.+?)\s*$/ ) {
            return uc($1);
        }
    }
    return '';
}