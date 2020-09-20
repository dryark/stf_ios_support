#!/usr/bin/perl -w
use strict;
use Data::Dumper;

my $main = `git log -1 --date=unix`;
my $mainT = 0;
if( $main =~ m/Date:\s+([0-9]+)/ ) {
    $mainT = $1;
}
my $arg = $ARGV[0];
if( -e "temp/check-ok-$mainT" && ( !$arg || $arg ne 'force' ) ) {
	print "Versions already checked; skipping version check\n";
    exit;
}

my $versions = `./get-version-info.sh --unix --wdasource`;
open( my $vfile, ">temp/current_versions.json" );
print $vfile $versions;
close( $vfile );
$versions =~ s/:/=>/g;
$versions =~ s/"/'/g;

my $ob = eval( $versions );

my $have_issues = 0;
my $reqs = {
    # h264_to_jpeg is no longer the primary video mechanism
    # h264_to_jpeg     => { min => 1588831486 },
    
    ios_video_stream => { min => 1597980831, message => "Then run `make cleanivs` them `make`" },
    wdaproxy         => { min => 1594408035, message => "Then run `make cleanwdaproxy` then `make`" },
    wda              => { min => 1596738353, message => "Then run `make cleanwda` them `make`", name => 'WebDriverAgent' },
    ios_avf_pull     => { min => 1597380907 },
    stf              => { min => 1597980993, name => 'stf-ios-provider' },
    device_trigger   => { min => 1578609558, name => 'osx_ios_device_trigger' }
};
for my $name ( keys %$reqs ) {
    my $repo = $ob->{ $name };
    if( !$repo ) {
    	$have_issues = 1;
        print "repos/$name is missing\n";
        next;
    }
    my $error = $repo->{error};
    if( $error ) {
    	$have_issues = 1;
		print "$name; error: $error\n";
		next;
	}
    my $remote = $repo->{remote};
    my $date = $repo->{date};
    my $dirname = $repo->{name} || $name;
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