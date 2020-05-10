#!/usr/bin/perl -w
use strict;
my $brew_check = `./util/brewser.pl checkdeps stf_ios_support.rb`;
if( $brew_check =~ m/Missing/ ) {
  print STDERR $brew_check, "\nRun init.sh to correct\n";
  print "x";
  exit(1);
}
if( $brew_check =~ m/Brew must be installed/ ) {
  print STDERR "Brew must be installed", "\nRun init.sh to correct\n";
  print "x";
  exit(1);
}
`./check-versions.pl`;
