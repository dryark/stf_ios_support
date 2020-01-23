#!/usr/bin/perl -w
use strict;
use Data::Dumper;
use IPC::Open2;
use File::Temp qw/tempfile/;

my $user = getlogin();
my $db = "/Users/$user/Library/Application Support/com.apple.TCC/TCC.db";
my @cols = qw/
  service
  client
  client_type
  allowed
  prompt_count
  csreq
  policy_id
  indirect_object_identifier
  indirect_object_code_identity
  flags
  last_modified
/;
  
my %usehex = (
  csreq => 1,
  indirect_object_code_identity => 1
);

my $action = $ARGV[0] || 'usage';
if( $action eq 'getcamera') {
  print_all_camera_access();
}
elsif( $action eq 'getcontrol' ) {
  print_all_control();
}
elsif( $action eq 'getall' ) {
  print_all_access();
}
elsif( $action eq 'addcamera' ) {
  my $app = $ARGV[1] || '/Applications/STF Coordinator.app';
  add_camera_access( $app );
}
elsif( $action eq 'delcamera' ) {
  my $app = $ARGV[1];
  del_camera_access( $app );
}
elsif( $action eq 'addcontrol' ) {
  my $app = $ARGV[1];
  my $app2 = $ARGV[2];
  add_control( $app, $app2 );
}
elsif( $action eq 'delcontrol' ) {
  my $app = $ARGV[1];
  my $app2 = $ARGV[2];
  del_control( $app, $app2 );
}
elsif( $action eq 'usage' ) {
  my $w = "\033[97m";
  my $o = "\033[0m";
  print <<DONE;
Usage:
  ./tcc.pl <action> [parameter(s) of action]
  
Actions:
  ${w}getcamera$o
    Show all of the apps with access to the camera
    
  ${w}getcontrol$o
    Show which apps can control which other apps
    
  ${w}getall$o
    Show all entries in the access DB
    
  ${w}addcamera$o <path to app>
    Give camera permission to the specified app
    Example: addcamera /Applications/Utilities/Terminal.app
    
  ${w}addcontrol$o <path to app1> <path to app2>
    Give app1 permission to control app2 using OSA / Applescript
    
  ${w}delcamera$o <path to app>
    Remove camera permissions for app
  
  ${w}delcontrol$o <path to app1> <path to app2>
    Remove permissions of app1 to control app2
DONE
}

sub get_access {
  my $where = shift;
  
  my $selectCols = "";
  for my $col ( @cols ) {
    if( $usehex{ $col } ) {
      $selectCols .= "hex($col) as $col,";
    }
    else {
      $selectCols .= "quote($col) as $col,";
    }
  }
  chop $selectCols;
  
  my $wheretext = '';
  
  if( $where ) {
    my @conds;
    for my $key ( keys %$where ) {
      my $val = $where->{ $key };
      push( @conds, "$key=$val" );
    }
    $wheretext = "where " . join( ',', @conds );
  }
  
  my $lines = `sqlite3 "$db" -line "select $selectCols from access $wheretext;"`;
  $lines .= "\n \n";
  my $row = {};
  my @rows;
  for my $line ( split( '\n', $lines ) ) {
    if( $line eq "" ) {
      push( @rows, $row );
      $row = {};
      next;
    }
    if( $line =~ m/([a-z()_]+) = (.+)$/ ) {
      my $name = $1;
      my $val = $2;
      if( $usehex{ $name } ) {
        my $raw = pack("H*", $val);
        $row->{ $name } = pipe_in_out( $raw, "csreq -r- -t" );
      }
      else {
        $row->{ $name } = $val;
      }
    }
  }
  return \@rows;
}

sub print_all_camera_access {
  my $rows = get_access( { service => "'kTCCServiceCamera'" } );
  #print Dumper( $rows );
  my $idents = get_app_idents();
  for my $row ( @$rows ) {
    print_camera_access( $idents, $row );
    print "\n";
  }
}

sub print_all_control {
  my $rows = get_access( { service => "'kTCCServiceAppleEvents\'" } );
  print Dumper( $rows );
  my $idents = get_app_idents();
  
  for my $row ( @$rows ) {
    print_control( $idents, $row );
    print "\n";
  }
}

sub print_control {
  my ( $idents, $row ) = @_;
  
  my $allowed = $row->{allowed};
  my $csreq = $row->{csreq};
  my $client = $row->{client};
  my $indir = $row->{indirect_object_identifier};
  my $indir_ident = $row->{indirect_object_code_identity};
  
  if( $allowed ) {
    print "Controlling App = $client\nControlling App Identity = $csreq\n";
    my $clientNoQuote = substr( $client, 1, -1 );
    if( $idents->{ $clientNoQuote } ) {
      my $full = $idents->{ $clientNoQuote };
      print "  Possible Match = $full\n";
      my $pIdent = app_to_ident( $full );
      if( $pIdent ) {
        print "  Possible Match Identity = $pIdent\n";
      }
    }
    
    print "Controlled App = $indir\nControlled App Identity = $indir_ident\n";
    my $indirNoQuote = substr( $indir, 1, -1 );
    if( $idents->{ $indirNoQuote } ) {
      my $full = $idents->{ $indirNoQuote };
      print "  Possible Match = $full\n";
      my $pIdent = app_to_ident( $full );
      if( $pIdent ) {
        print "  Possible Match Identity = $pIdent\n";
      }
    }
  }
}

sub app_to_ident {
  my $app = shift;
  my @matchLines = `codesign -d -r- "$app" 2>/dev/null`;
  for my $line ( @matchLines ) {
    if( $line =~ m/designated => (.+)/ ) {
      return $1;
    }
  }
  return "";
}

sub print_all_access {
  my $rows = get_access( 0 );
  print Dumper( $rows );
  
}

sub print_camera_access {
  my ( $idents, $row ) = @_;
  my $allowed = $row->{allowed};
  my $csreq = $row->{csreq};
  my $client = $row->{client};
  
  if( $allowed ) {
    print "CS Req = $csreq\nClient = $client\n";
    my $clientNoQuote = substr( $client, 1, -1 );
    if( $idents->{ $clientNoQuote } ) {
      my $full = $idents->{ $clientNoQuote };
      print "  Possible Match = $full\n";
      my $pIdent = app_to_ident( $full );
      if( $pIdent ) {
        print "  Possible Match Identity = $pIdent\n";
      }
    }
  }
}

sub pipe_in_out {
  my ( $in, $cmd ) = @_;
  
  my $out = '';
  my $pid = open2( \*SUB_OUT, \*SUB_IN, $cmd );
  
  print SUB_IN "$in\cD";
  
  while( <SUB_OUT> ) {
    $out .= $_;
  }
  chomp $out;
  
  waitpid( $pid, 0 );
  
  return $out;
}

sub get_app_idents {
  my %idents;
  
  opendir( my $dh, "/Applications" );
  my @files = readdir( $dh );
  closedir( $dh );
  for my $file ( @files ) {
    next if( $file =~ m/^\.+$/ );
    next if( $file !~ m/\.app$/ );
    my $full = "/Applications/$file";
    my $ident = get_app_bundle( $full );
    if( $ident ) {
      $idents{ $ident } = $full;
    }
  }
  return \%idents;
}

sub get_app_bundle {
  my $full = shift;
  my @lines = `plutil -extract CFBundleIdentifier xml1 "$full/Contents/Info.plist" -o -`;
  for my $line ( @lines ) {
    if( $line =~ m|<string>(.+)</string>| ) {
      return $1;
    }
  }
  return "";
}

sub add_camera_access {
  my $app = shift;
  my $appIdent = app_to_ident( $app );
  #print "Ident: $appIdent\n";
  my $hex = ident_to_csreq( $appIdent );
  #print "$hex\n";
  
  my $bundle = get_app_bundle( $app );
  #print "Bundle: $bundle\n";
  
  `sqlite3 "$db" "delete from access where service='kTCCServiceCamera' and client='$bundle'"`;
  sql_insert( 'access', {
    service      => "'kTCCServiceCamera'",
    client       => "'$bundle'",
    client_type  => 0,
    allowed      => 1,
    prompt_count => 1,
    csreq        => "x'$hex'",
    policy_id    => "'NULL'",
    indirect_object_identifier => "'UNUSED'",
    flags        => 0
  } );
}

sub del_camera_access {
  my $app = shift;
  my $bundle = get_app_bundle( $app );
  `sqlite3 "$db" "delete from access where service='kTCCServiceCamera' and client='$bundle'"`;
}

sub ident_to_csreq {
  my $ident = shift;
  
  open( my $fh, "| csreq -r- -b ./test" );
  print $fh $ident;
  close( $fh );
  open( my $bh, "<./test" );
  binmode( $bh );
  my $data;
  {
    local $/ = undef;
    $data = <$bh>;
  }
  close( $bh );
  my $hex = uc( unpack("H*", $data) );
  
  return $hex;
}

sub add_control {
  my ( $app, $app2 ) = @_;
  my $appIdent = app_to_ident( $app );
  my $app2Ident = app_to_ident( $app2 );
  #print "Ident: $appIdent\n";
  my $hex = ident_to_csreq( $appIdent );
  my $hex2 = ident_to_csreq( $app2Ident );
  #print "$hex\n";
  
  my $bundle = get_app_bundle( $app );
  my $bundle2 = get_app_bundle( $app2 );
  #print "Bundle: $bundle\n";
  
  `sqlite3 "$db" "delete from access where service='kTCCServiceAppleEvents' and client='$bundle' and indirect_object_identifier='$bundle2'"`;
  sql_insert( 'access', {
    service      => "'kTCCServiceAppleEvents'",
    client       => "'$bundle'",
    client_type  => 0,
    allowed      => 1,
    prompt_count => 1,
    csreq        => "x'$hex'",
    policy_id    => "'NULL'",
    indirect_object_identifier => "'$bundle2'",
    indirect_object_code_identity => "x'$hex2'",
    flags        => "NULL"
  } );
}

sub del_control {
  my ( $app, $app2 ) = @_;
  my $bundle = get_app_bundle( $app );
  my $bundle2 = get_app_bundle( $app2 );
  
  `sqlite3 "$db" "delete from access where service='kTCCServiceAppleEvents' and client='$bundle' and indirect_object_identifier='$bundle2'"`;
}

sub sql_insert {
  my ( $table, $vals ) = @_;
  
  my @keys = sort keys %$vals;
  
  my @valset;
  for my $key ( @keys ) {
    my $val = $vals->{ $key };
    push( @valset, $val );
  }
  my $keytext = join(',', @keys );
  my $valtext = join(',', @valset );
  `sqlite3 "$db" "insert into $table ($keytext) values($valtext)"`;
}