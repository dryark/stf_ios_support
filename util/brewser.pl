#!/usr/bin/perl -w
use strict;
use JSON::PP qw/decode_json/;
use Data::Dumper;

my $GR="\033[32m";
my $RED="\033[91m";
my $RST="\033[0m";
my $action = $ARGV[0] || 'help';

if( !`which brew` ) {
  print "Brew must be installed\n";
  help();
  exit(1);
}

if( $action eq 'list' ) {
  my $pkgs = get_pkg_versions();
  for my $pkg ( keys %$pkgs ) {
    my $ver = $pkgs->{$pkg};
    print "$pkg,$ver\n";
  }
}
elsif( $action eq 'installdeps' ) {
  my $rbspec = $ARGV[1] or die "Ruby spec file must be given";
  my $spec = read_file( $rbspec );
  my $pkgs = get_pkg_versions();
  my @need;
  for my $line ( split( "\n", $spec ) ) {
    if( $line =~ m/^\s*depends_on "(.+?)"/ ) {
      my $dep = $1;
      if( my $ver = $pkgs->{ $dep } ) {
        print "$GR$dep\t\t=> version $ver$RST\n";
      }
      else {
        push( @need, $dep );
      }
    }
  }
  if( @need ) {
    my $allneed = join(' ',@need);
    print "Installing missing packages:\n";
    print "  ".join("\n  ",@need);
    `brew install $allneed 1>&2`;
  }
}
elsif( $action eq 'checkdeps' ) {
  my $rbspec = $ARGV[1] or die "Ruby spec file must be given";
  my $spec = read_file( $rbspec );
  my $pkgs = get_pkg_versions();
  my @need;
  for my $line ( split( "\n", $spec ) ) {
    if( $line =~ m/^\s*depends_on "(.+?)"/ ) {
      my $dep = $1;
      if( my $ver = $pkgs->{ $dep } ) {
        print "$GR$dep\t\t=> version $ver$RST\n";
      }
      else {
        push( @need, $dep );
      }
    }
  }
  if( @need ) {
    my $allneed = join(' ',@need);
    print "Missing brew package(s):\n";
    print "  ".join("\n  ",@need);
  }
}
elsif( $action eq 'info' ) {
  my $pkg = $ARGV[1];
  my $info = install_info( $pkg );
  if( !$info ) {
    print "$pkg is not installed\n";
    exit 1;
  }
  print JSON::PP->new->ascii->pretty->encode( $info );
}
elsif( $action eq 'ensurehead' ) {
  ensure_head( $ARGV[1] );
}
else {
  help();
}

sub help {
  print "Brewser
  Usage:
    ./brewser.pl [action] [args]
  Actions:
    list - list packages and versions installed
    info [package name] - pretty print json install receipt of named package
    ensurehead [package name] - ensure HEAD version of a package is installed
      If a non-HEAD version is installed, it will be removed and the current HEAD installed.
      If a HEAD version is installed, even if old, nothing will happen.
    installdeps [ruby spec file] - install dependencies for a specified brew package spec file\n";
}
sub get_pkg_versions {
  my %pkgs;
  my @dirs = sort `find /usr/local/Cellar -name .brew -maxdepth 3 -type d`;
  for my $dir ( @dirs ) {
    $pkgs{ $1 } = $2 if( $dir =~ m|^/usr/local/Cellar/([^/]+)/([^/]+)/\.brew$| );
  }
  return \%pkgs;
}

sub read_file {
  open( my $fh, "<".shift );
  my $data;
  { local $/ = undef; $data = <$fh>; }
  close( $fh );
  return $data;
}

sub install_info {
  my ( $pkg, $ver ) = @_;
  if( !$ver ) {
    my $path = `find /usr/local/Cellar/$pkg -maxdepth 1 2>/dev/null | tail -1`;
    chomp $path;
    return 0 if( !$path );
    my @parts = split( "/", $path );
    $ver = pop @parts;
  }
  return decode_json( read_file( "/usr/local/Cellar/$pkg/$ver/INSTALL_RECEIPT.json" ) );
}

sub ensure_head {
  my $pkg = shift;
  my $info = install_info( $pkg );
  my $spec = $info ? $info->{source}{spec} : '';
  if( !$spec || $spec ne 'head' ) {
    print "$pkg - Installing HEAD\n";
    `brew uninstall $pkg --ignore-dependencies` if( $spec );
    `brew install --HEAD $pkg`;
  }
  else {
    print "$GR$pkg - HEAD already installed$RST\n";
  }
}