#!/usr/bin/perl -w
use strict;
use File::Copy;

my $configSrc = $ENV{CONFIG_SRC} or die "ENV CONFIG_SRC not set";
my $installDir = $ENV{INSTALL_DIR} or die "ENV INSTALL_DIR not set";
my $updateDir = $ENV{UPDATE_DIR} or die "ENV UPDATE_DIR not set";

my $dist = $ARGV[0];

print "Extracting $dist to $installDir\n";

if( ! -e "/usr/local/bin/pv" ) {
	print "pv not installed. installing...\n";
	system("/usr/local/bin/brew install pv");
}

print STDERR "PROGSTART\n";
system("/usr/local/bin/pv -n $updateDir/$dist | /usr/bin/tar -xf - -C $installDir");
print STDERR "PROGEND\n";

my $configDest = "$installDir/config.json";
print "Copying $configSrc to $configDest\n";
copy( $configSrc, $configDest );

chdir $installDir;
$ENV{'PATH'} = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin";
system('./init.sh');