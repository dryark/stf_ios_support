#!/usr/bin/perl -w
use strict;

my $hostname = "localhost";
my $mainip = "127.0.0.1";

my $template = "runnercert.tmpl";
my $template_data = slurp( $template );

$template_data =~ s/HOSTNAME/$hostname/g;
$template_data =~ s/IPADDR/$mainip/g;

open( my $outfh, ">runnercert.conf" );
print $outfh $template_data;
my $ips = get_ips();
my $index = 2;
for my $ip ( @$ips ) {
    print $outfh "DNS.$index   = $ip\n";
    $index++;
}
close( $outfh );

`openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout server.key -out server.crt -config runnercert.conf -subj "/C=US/ST=Washington/L=Seattle/O=Dis/CN=$hostname"`;

sub slurp {
	my $file = shift;
	open( my $fh, "<$file" );
	my $data;
	{
		local $/ = undef;
		$data = <$fh>;
	}
	close( $fh );
	return $data;
}

sub get_ips {
    my @lines = `ifconfig`;
    my @ips;
    for my $line ( @lines ) {
        next if( $line !~ m/inet / );
        if( $line =~ m/inet ([0-9.]+) / ) {
            push( @ips, $1 );
        }
    }
    return \@ips;
}