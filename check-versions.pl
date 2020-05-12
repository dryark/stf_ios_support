#!/usr/bin/perl -w
use strict;
use Data::Dumper;

my $main = `git log -1 --date=unix`;
my $mainT = 0;
if( $main =~ m/Date:\s+([0-9]+)/ ) {
    $mainT = $1;
}
if( -e "temp/check-ok-$mainT" ) {
    exit;
}

my $versions = `./get-version-info.sh --unix --wdasource`;
$versions =~ s/:/=>/g;
$versions =~ s/"/'/g;

my $ob = eval( $versions );

my $have_issues = 0;
my $reqs = {
    h264_to_jpeg     => { min => 1588831486 },
    ios_video_stream => { min => 1589311851, message => "Then run `make cleanivs` them `make`" },
    wdaproxy         => { min => 1589245810, message => "Then run `make cleanwdaproxy` then `make`" },
    wda              => { min => 1588413226, message => "Then run `make cleanwda` them `make`" }
};
for my $name ( keys %$reqs ) {
    my $repo = $ob->{ $name };
    if( !$repo ) {
        print "repos/$name is missing\n";
        next;
    }
    my $remote = $repo->{remote};
    my $date = $repo->{date};
    $remote =~ s/=>/:/;
    my $req = $reqs->{ $name };
    if( $req->{ min } ) {
        my $min = $req->{ min };
        if( $date < $min ) {
            my $msg = $req->{ message } || '';
            print STDERR "repos/$name is out of date. Please git pull it. $msg\n";
            $have_issues = 1;
        }
    }
}
if( !$have_issues ) {
    open( my $fh, ">temp/check-ok-$mainT" );
    print $fh 1;
    close( $fh );
}